package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/cors"
	"github.com/xenitab/azad-kube-proxy/pkg/handlers"
	"github.com/xenitab/azad-kube-proxy/pkg/health"
	"github.com/xenitab/azad-kube-proxy/pkg/metrics"
	"github.com/xenitab/azad-kube-proxy/pkg/user"
	"golang.org/x/sync/errgroup"
)

// ClientInterface ...
type ClientInterface interface {
	Start(ctx context.Context) error
	listenAndServe(httpServer *http.Server) error
	getHTTPServer(handler http.Handler) *http.Server
	getReverseProxy(ctx context.Context) *httputil.ReverseProxy
	getProxyTransport() *http.Transport
}

// Client ...
type Client struct {
	Config        config.Config
	CacheClient   cache.ClientInterface
	UserClient    user.ClientInterface
	AzureClient   azure.ClientInterface
	MetricsClient metrics.ClientInterface
	HealthClient  health.ClientInterface
	CORSClient    cors.ClientInterface
}

// NewProxyClient ...
func NewProxyClient(ctx context.Context, config config.Config) (ClientInterface, error) {
	cacheClient, err := cache.NewCache(ctx, config.CacheEngine, config)
	if err != nil {
		return nil, err
	}

	azureClient, err := azure.NewAzureClient(ctx, config.ClientID, config.ClientSecret, config.TenantID, config.AzureADGroupPrefix, cacheClient)
	if err != nil {
		return nil, err
	}

	userClient := user.NewUserClient(config, azureClient)

	metricsClient, err := metrics.NewMetricsClient(ctx, config)
	if err != nil {
		return nil, err
	}

	healthClient, err := health.NewHealthClient(ctx, config, azureClient)
	if err != nil {
		return nil, err
	}

	corsClient := cors.NewCORSClient(config)

	proxyClient := Client{
		Config:        config,
		CacheClient:   cacheClient,
		UserClient:    userClient,
		AzureClient:   azureClient,
		MetricsClient: metricsClient,
		HealthClient:  healthClient,
		CORSClient:    corsClient,
	}

	return &proxyClient, nil
}

// Start launches the reverse proxy
func (client *Client) Start(ctx context.Context) error {
	log := logr.FromContextOrDiscard(ctx)

	// Signal handler
	stopChan := make(chan os.Signal, 2)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGPIPE)

	// Error group
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	// Initiate group sync
	log.Info("Starting group sync")
	syncTicker, syncChan, err := client.AzureClient.StartSyncGroups(ctx, client.Config.GroupSyncInterval)
	if err != nil {
		return err
	}
	var stopGroupSync func() = func() {
		syncTicker.Stop()
		syncChan <- true
	}
	defer stopGroupSync()

	// Configure reverse proxy and http server
	proxyHandlers, err := handlers.NewHandlersClient(ctx, client.Config, client.CacheClient, client.UserClient, client.HealthClient)
	if err != nil {
		return err
	}
	log.Info("Initializing reverse proxy", "ListenerAddress", client.Config.ListenerAddress, "MetricsListenerAddress", client.Config.MetricsListenerAddress, "ListenerTLSConfig.Enabled", client.Config.ListenerTLSConfig.Enabled)
	proxy := client.getReverseProxy(ctx)
	proxy.ErrorHandler = proxyHandlers.ErrorHandler(ctx)

	// Setup metrics router
	metricsRouter := mux.NewRouter()

	metricsRouter.HandleFunc("/readyz", proxyHandlers.ReadinessHandler(ctx)).Methods("GET")
	metricsRouter.HandleFunc("/healthz", proxyHandlers.LivenessHandler(ctx)).Methods("GET")

	metricsRouter, err = client.MetricsClient.MetricsHandler(ctx, metricsRouter)
	if err != nil {
		return err
	}

	metricsHttpServer := client.getHTTPMetricsServer(metricsRouter)

	// Start metrics server
	g.Go(func() error {
		err := client.listenAndServe(metricsHttpServer)
		if err != nil && err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	// Setup http router
	router := mux.NewRouter()

	oidcHandler := handlers.NewOIDCHandler(proxyHandlers.AzadKubeProxyHandler(ctx, proxy), client.Config.TenantID, client.Config.ClientID)

	router.PathPrefix("/").Handler(oidcHandler)

	router.Use(client.CORSClient.Middleware)

	httpServer := client.getHTTPServer(router)

	// Start HTTP server
	g.Go(func() error {
		err := client.listenAndServe(httpServer)
		if err != nil && err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	log.Info("Server started")

	// Blocks until signal is sent
	var doneMsg string
	select {
	case sig := <-stopChan:
		doneMsg = fmt.Sprintf("os.Signal (%s)", sig)
	case <-ctx.Done():
		doneMsg = "context"
	}

	log.Info("Server shutdown initiated", "reason", doneMsg)

	// Shutdown http server
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	g.Go(func() error {
		err = httpServer.Shutdown(shutdownCtx)
		if err != nil {
			log.Error(err, "http server shutdown failed")
			return err
		}

		return nil
	})

	// Shutdown metrics server
	g.Go(func() error {
		err = metricsHttpServer.Shutdown(shutdownCtx)
		if err != nil {
			log.Error(err, "metrics server shutdown failed")
			return err
		}

		return nil
	})

	err = g.Wait()
	if err != nil {
		return fmt.Errorf("error groups error: %w", err)
	}

	log.Info("Server exited properly")

	return nil
}

func (client *Client) listenAndServe(httpServer *http.Server) error {
	if client.Config.ListenerTLSConfig.Enabled {
		return httpServer.ListenAndServeTLS(client.Config.ListenerTLSConfig.CertificatePath, client.Config.ListenerTLSConfig.KeyPath)
	}

	return httpServer.ListenAndServe()
}

func (client *Client) getHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              client.Config.ListenerAddress,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func (client *Client) getHTTPMetricsServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              client.Config.MetricsListenerAddress,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func (client *Client) getReverseProxy(ctx context.Context) *httputil.ReverseProxy {
	reverseProxy := httputil.NewSingleHostReverseProxy(client.Config.KubernetesConfig.URL)
	reverseProxy.Transport = client.getProxyTransport()
	return reverseProxy
}

func (client *Client) getProxyTransport() *http.Transport {
	return &http.Transport{
		TLSClientConfig: getProxyTLSClientConfig(client.Config.KubernetesConfig.ValidateCertificate, client.Config.KubernetesConfig.RootCA),
	}
}

func getProxyTLSClientConfig(validateCertificate bool, rootCA *x509.CertPool) *tls.Config {
	if !validateCertificate {
		return &tls.Config{InsecureSkipVerify: true} // #nosec
	}

	return &tls.Config{InsecureSkipVerify: false, RootCAs: rootCA} // #nosec
}
