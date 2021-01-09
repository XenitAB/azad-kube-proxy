package proxy

import (
	"context"
	"crypto/tls"
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
	// Initiate memory cache
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

	userClient := user.NewUserClient(ctx, config, cache, azureClient)

	rp := Server{
		Config:       config,
		Cache:        cache,
		OIDCVerifier: oidcVerifier,
		UserClient:   userClient,
	}

	return &rp, nil
}

// Start launches the reverse proxy
func (server *Server) Start(ctx context.Context) error {
	log := logr.FromContext(ctx)
	config := server.Config

	// Signal handler
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	if config.KubernetesConfig.ValidateCertificate {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            config.KubernetesConfig.RootCA,
		}
	}

	// Configure revers proxy and http server
	log.Info("Initializing reverse proxy", "ListenerAddress", config.ListenerAddress)
	proxy := httputil.NewSingleHostReverseProxy(config.KubernetesConfig.URL)
	proxy.ErrorHandler = server.errorHandler(ctx)
	proxy.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	router := mux.NewRouter()
	router.HandleFunc("/readyz", server.readinessHandler(ctx)).Methods("GET")
	router.HandleFunc("/healthz", server.livenessHandler(ctx)).Methods("GET")

	// Initiate Azure AD group sync
	syncTicker, syncChan, err := server.UserClient.AzureClient.StartSyncTickerAzureADGroups(ctx, 5*time.Minute)
	if err != nil {
		return err
	}

	router.PathPrefix("/").HandlerFunc(server.proxyHandler(ctx, proxy))
	srv := &http.Server{Addr: config.ListenerAddress, Handler: router}

	// Start HTTP server
	go func() {
		switch config.ListenerTLSConfig.Enabled {
		case true:
			if err := srv.ListenAndServeTLS(config.ListenerTLSConfig.CertificatePath, config.ListenerTLSConfig.KeyPath); err != nil && err != http.ErrServerClosed {
				log.Error(err, "Http Server Error")
			}
		case false:
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error(err, "Http Server Error")
			}
		}

	}()
	log.Info("Server started")

	// Blocks until singal is sent
	<-done
	syncTicker.Stop()
	syncChan <- true
	log.Info("Server stopped")

	// Shutdown http server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error(err, "Server shutdown failed")
		return err
	}

	log.Info("Server exited properly")
	return nil
}
