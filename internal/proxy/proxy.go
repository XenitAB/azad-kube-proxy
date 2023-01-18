package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/internal/config"
	"github.com/xenitab/azad-kube-proxy/internal/metrics"
	"golang.org/x/sync/errgroup"
)

type Proxy interface {
	Start(ctx context.Context) error
	listenAndServe(httpServer *http.Server) error
	getHTTPServer(handler http.Handler) *http.Server
	getReverseProxy(ctx context.Context) *httputil.ReverseProxy
	getProxyTransport() *http.Transport
}

type proxy struct {
	cache         Cache
	user          User
	azure         Azure
	MetricsClient metrics.ClientInterface
	health        Health
	cors          Cors

	cfg              *config.Config
	kubernetesURL    *url.URL
	kubernetesRootCA *x509.CertPool
}

func New(ctx context.Context, cfg *config.Config) (*proxy, error) {
	cacheClient, err := newMemoryCache(time.Duration(cfg.GroupSyncInterval) * time.Minute)
	if err != nil {
		return nil, err
	}

	azureClient, err := newAzureClient(ctx, cfg.AzureClientID, cfg.AzureClientSecret, cfg.AzureTenantID, cfg.AzureADGroupPrefix, cacheClient)
	if err != nil {
		return nil, err
	}

	userClient := newUser(cfg, azureClient)

	metricsClient, err := metrics.NewMetricsClient(ctx, cfg)
	if err != nil {
		return nil, err
	}

	healthClient, err := newHealthClient(ctx, cfg, azureClient)
	if err != nil {
		return nil, err
	}

	corsClient := newCors(cfg)

	kubernetesURL, err := getKubernetesAPIUrl(cfg.KubernetesAPIHost, cfg.KubernetesAPIPort, cfg.KubernetesAPITLS)
	if err != nil {
		return nil, err
	}

	kubernetesRootCA, err := getCertificate(ctx, cfg.KubernetesAPICACertPath)
	if err != nil {
		return nil, err
	}

	proxyClient := proxy{
		cache:            cacheClient,
		user:             userClient,
		azure:            azureClient,
		MetricsClient:    metricsClient,
		health:           healthClient,
		cors:             corsClient,
		cfg:              cfg,
		kubernetesURL:    kubernetesURL,
		kubernetesRootCA: kubernetesRootCA,
	}

	return &proxyClient, nil
}

// Start launches the reverse proxy
func (p *proxy) Start(ctx context.Context) error {
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
	syncTicker, syncChan, err := p.azure.StartSyncGroups(ctx, time.Duration(p.cfg.GroupSyncInterval)*time.Minute)
	if err != nil {
		return err
	}
	var stopGroupSync func() = func() {
		syncTicker.Stop()
		syncChan <- true
	}
	defer stopGroupSync()

	// Configure reverse proxy and http server
	proxyHandlers, err := newHandlers(ctx, p.cfg, p.cache, p.user, p.health)
	if err != nil {
		return err
	}
	log.Info("Initializing reverse proxy", "ListenerAddress", p.cfg.ListenerAddress, "MetricsListenerAddress", p.cfg.MetricsListenerAddress, "ListenerTLSConfigEnabled", p.cfg.ListenerTLSConfigEnabled)
	proxy := p.getReverseProxy(ctx)
	proxy.ErrorHandler = proxyHandlers.error(ctx)

	// Setup metrics router
	metricsRouter := mux.NewRouter()

	metricsRouter.HandleFunc("/readyz", proxyHandlers.readiness(ctx)).Methods("GET")
	metricsRouter.HandleFunc("/healthz", proxyHandlers.liveness(ctx)).Methods("GET")

	metricsRouter, err = p.MetricsClient.MetricsHandler(ctx, metricsRouter)
	if err != nil {
		return err
	}

	metricsHttpServer := p.getHTTPMetricsServer(metricsRouter)

	// Start metrics server
	g.Go(func() error {
		err := p.listenAndServe(metricsHttpServer)
		if err != nil && err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	// Setup http router
	router := mux.NewRouter()

	oidcHandler := newOIDCHandler(proxyHandlers.proxy(ctx, proxy), p.cfg.AzureTenantID, p.cfg.AzureClientID)

	router.PathPrefix("/").Handler(oidcHandler)

	router.Use(p.cors.Middleware)

	httpServer := p.getHTTPServer(router)

	// Start HTTP server
	g.Go(func() error {
		err := p.listenAndServe(httpServer)
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

func (client *proxy) listenAndServe(httpServer *http.Server) error {
	if client.cfg.ListenerTLSConfigEnabled {
		return httpServer.ListenAndServeTLS(client.cfg.ListenerTLSConfigCertificatePath, client.cfg.ListenerTLSConfigKeyPath)
	}

	return httpServer.ListenAndServe()
}

func (client *proxy) getHTTPServer(handler http.Handler) *http.Server {
	addr := fmt.Sprintf("%s:%d", client.cfg.ListenerAddress, client.cfg.ListenerPort)
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func (client *proxy) getHTTPMetricsServer(handler http.Handler) *http.Server {
	addr := fmt.Sprintf("%s:%d", client.cfg.MetricsListenerAddress, client.cfg.MetricsListenerPort)
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func (client *proxy) getReverseProxy(ctx context.Context) *httputil.ReverseProxy {
	reverseProxy := httputil.NewSingleHostReverseProxy(client.kubernetesURL)
	reverseProxy.Transport = client.getProxyTransport()
	return reverseProxy
}

func (client *proxy) getProxyTransport() *http.Transport {
	return &http.Transport{
		TLSClientConfig: getProxyTLSClientConfig(client.cfg.KubernetesAPIValidateCert, client.kubernetesRootCA),
	}
}

func getProxyTLSClientConfig(validateCertificate bool, rootCA *x509.CertPool) *tls.Config {
	if !validateCertificate {
		return &tls.Config{InsecureSkipVerify: true} // #nosec
	}

	return &tls.Config{InsecureSkipVerify: false, RootCAs: rootCA} // #nosec
}

func getKubernetesAPIUrl(host string, port int, tls bool) (*url.URL, error) {
	httpScheme := getHTTPScheme(tls)
	return url.Parse(fmt.Sprintf("%s://%s:%d", httpScheme, host, port))
}

func getHTTPScheme(tls bool) string {
	if tls {
		return "https"
	}

	return "http"
}
