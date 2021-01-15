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

	oidc "github.com/coreos/go-oidc"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/claims"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/user"
)

// Server ...
type Server struct {
	Config       config.Config
	Cache        cache.Cache
	OIDCVerifier *oidc.IDTokenVerifier
	UserClient   *user.Client
}

// NewProxyServer returns a proxy client or an error
func NewProxyServer(ctx context.Context, config config.Config) (*Server, error) {
	cache, err := cache.NewCache(ctx, config.CacheEngine, config)
	if err != nil {
		return nil, err
	}

	oidcVerifier, err := claims.GetOIDCVerifier(ctx, config.TenantID, config.ClientID)
	if err != nil {
		return nil, err
	}

	azureClient, err := azure.NewAzureClient(ctx, config.ClientID, config.ClientSecret, config.TenantID, config.AzureADGroupPrefix, cache)
	if err != nil {
		return nil, err
	}

	userClient := user.NewUserClient(config, azureClient)

	proxyServer := Server{
		Config:       config,
		Cache:        cache,
		OIDCVerifier: oidcVerifier,
		UserClient:   userClient,
	}

	return &proxyServer, nil
}

// Start launches the reverse proxy
func (server *Server) Start(ctx context.Context) error {
	log := logr.FromContext(ctx)

	// Signal handler
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Initiate group sync
	log.Info("Starting group sync")
	syncTicker, syncChan, err := server.UserClient.AzureClient.StartSyncGroups(ctx, 5*time.Minute)
	if err != nil {
		return err
	}
	var stopGroupSync func() = func() {
		// Stop group sync
		syncTicker.Stop()
		syncChan <- true
		return
	}
	defer stopGroupSync()

	// Configure reverse proxy and http server
	log.Info("Initializing reverse proxy", "ListenerAddress", server.Config.ListenerAddress)
	proxy := server.getReverseProxy(ctx)
	router := mux.NewRouter()
	router.HandleFunc("/readyz", server.readinessHandler(ctx)).Methods("GET")
	router.HandleFunc("/healthz", server.livenessHandler(ctx)).Methods("GET")
	router.PathPrefix("/").HandlerFunc(server.azadKubeProxyHandler(ctx, proxy))
	httpServer := server.getHTTPServer(router)

	// Start HTTP server
	go func() {
		err := server.listenAndServe(httpServer)
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

func (server *Server) listenAndServe(httpServer *http.Server) error {
	if server.Config.ListenerTLSConfig.Enabled {
		return httpServer.ListenAndServeTLS(server.Config.ListenerTLSConfig.CertificatePath, server.Config.ListenerTLSConfig.KeyPath)
	}

	return httpServer.ListenAndServe()
}

func (server *Server) getHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{Addr: server.Config.ListenerAddress, Handler: handler}
}

func (server *Server) getReverseProxy(ctx context.Context) *httputil.ReverseProxy {
	reverseProxy := httputil.NewSingleHostReverseProxy(server.Config.KubernetesConfig.URL)
	reverseProxy.ErrorHandler = server.errorHandler(ctx)
	reverseProxy.Transport = server.getProxyTransport()
	return reverseProxy
}

func (server *Server) getProxyTransport() *http.Transport {
	return &http.Transport{
		TLSClientConfig: getProxyTLSClientConfig(server.Config.KubernetesConfig.ValidateCertificate, server.Config.KubernetesConfig.RootCA),
	}
}

func getProxyTLSClientConfig(validateCertificate bool, rootCA *x509.CertPool) *tls.Config {
	if !validateCertificate {
		return &tls.Config{InsecureSkipVerify: true}
	}

	return &tls.Config{InsecureSkipVerify: false, RootCAs: rootCA}
}
