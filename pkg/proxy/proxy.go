package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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
	"github.com/xenitab/azad-kube-proxy/pkg/claims"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/dashboard"
	"github.com/xenitab/azad-kube-proxy/pkg/handlers"
	"github.com/xenitab/azad-kube-proxy/pkg/user"
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
	Config          config.Config
	CacheClient     cache.ClientInterface
	UserClient      user.ClientInterface
	AzureClient     azure.ClientInterface
	ClaimsClient    claims.ClientInterface
	DashboardClient dashboard.ClientInterface
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
	claimsClient := claims.NewClaimsClient()

	dashboardClient, err := dashboard.NewDashboardClient(ctx, config)
	if err != nil {
		return nil, err
	}

	proxyClient := Client{
		Config:          config,
		CacheClient:     cacheClient,
		UserClient:      userClient,
		AzureClient:     azureClient,
		ClaimsClient:    claimsClient,
		DashboardClient: dashboardClient,
	}

	return &proxyClient, nil
}

// Start launches the reverse proxy
func (client *Client) Start(ctx context.Context) error {
	log := logr.FromContext(ctx)

	// Signal handler
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Initiate group sync
	log.Info("Starting group sync")
	syncTicker, syncChan, err := client.AzureClient.StartSyncGroups(ctx, 5*time.Minute)
	if err != nil {
		return err
	}
	var stopGroupSync func() = func() {
		syncTicker.Stop()
		syncChan <- true
	}
	defer stopGroupSync()

	// Configure reverse proxy and http server
	proxyHandlers, err := handlers.NewHandlersClient(ctx, client.Config, client.CacheClient, client.UserClient, client.ClaimsClient)
	if err != nil {
		return err
	}
	log.Info("Initializing reverse proxy", "ListenerAddress", client.Config.ListenerAddress)
	proxy := client.getReverseProxy(ctx)
	proxy.ErrorHandler = proxyHandlers.ErrorHandler(ctx)

	router := mux.NewRouter()
	router.HandleFunc("/readyz", proxyHandlers.ReadinessHandler(ctx)).Methods("GET")
	router.HandleFunc("/healthz", proxyHandlers.LivenessHandler(ctx)).Methods("GET")
	routerWithDashboard := client.DashboardClient.DashboardHandler(ctx, router)
	routerWithDashboard.PathPrefix("/").HandlerFunc(proxyHandlers.AzadKubeProxyHandler(ctx, proxy))
	httpServer := client.getHTTPServer(routerWithDashboard)

	// Start HTTP server
	go func() {
		err := client.listenAndServe(httpServer)
		if err != nil && err != http.ErrServerClosed {
			log.Error(err, "Server error")
		}
	}()

	log.Info("Server started")

	// Blocks until signal is sent
	<-done

	log.Info("Server shutdown initiated")

	// Shutdown http server
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer func() {
		cancel()
	}()

	err = httpServer.Shutdown(shutdownCtx)
	if err != nil {
		log.Error(err, "Server shutdown failed")
		return err
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
	return &http.Server{Addr: client.Config.ListenerAddress, Handler: handler}
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
