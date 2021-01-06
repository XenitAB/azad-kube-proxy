package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
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
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/user"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// Proxy ...
type Proxy struct {
	Context      context.Context
	Config       config.Config
	Cache        cache.Client
	OIDCVerifier *oidc.IDTokenVerifier
	UserClient   user.User
}

// Start launches the reverse proxy
func Start(ctx context.Context, config config.Config) error {
	log := logr.FromContext(ctx)

	rp, err := newProxyClient(ctx, config)
	if err != nil {
		return err
	}

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
	proxy.ErrorHandler = rp.errorHandler(ctx)
	proxy.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	router := mux.NewRouter()
	router.HandleFunc("/readyz", rp.readinessHandler()).Methods("GET")
	router.HandleFunc("/healthz", rp.livenessHandler()).Methods("GET")

	// Initiate Azure AD group sync
	syncTicker, syncChan, err := rp.UserClient.AzureClient.StartSyncTickerAzureADGroups(5 * time.Minute)
	if err != nil {
		return err
	}

	router.PathPrefix("/").HandlerFunc(rp.proxyHandler(proxy))
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

func newProxyClient(ctx context.Context, config config.Config) (Proxy, error) {
	// Initiate memory cache
	var c cache.Client

	switch config.CacheEngine {
	case models.RedisCacheEngine:
		c = &cache.RedisCache{
			Address:    "127.0.0.1:6379",
			Password:   "",
			Database:   0,
			Context:    ctx,
			Expiration: 5 * time.Minute,
		}

		c.NewCache()
	case models.MemoryCacheEngine:
		c = &cache.MemoryCache{
			DefaultExpiration: 5 * time.Minute,
			CleanupInterval:   10 * time.Minute,
		}

		c.NewCache()
	}

	oidcVerifier, err := getOIDCVerifier(ctx, config)
	if err != nil {
		return Proxy{}, err
	}

	azureClient, err := azure.NewAzureClient(ctx, config.ClientID, config.ClientSecret, config.TenantID, config.AzureADGroupPrefix, c)
	if err != nil {
		return Proxy{}, err
	}

	userClient := user.NewUserClient(ctx, config, c, azureClient)

	rp := Proxy{
		Context:      ctx,
		Config:       config,
		Cache:        c,
		OIDCVerifier: oidcVerifier,
		UserClient:   userClient,
	}

	return rp, nil
}

func getOIDCVerifier(ctx context.Context, config config.Config) (*oidc.IDTokenVerifier, error) {
	log := logr.FromContext(ctx)
	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", config.TenantID)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		log.Error(err, "Unable to initiate OIDC provider")
		return nil, err
	}

	oidcConfig := &oidc.Config{
		ClientID: config.ClientID,
	}

	verifier := provider.Verifier(oidcConfig)

	return verifier, nil

}

// Inspiration: https://github.com/jetstack/kube-oidc-proxy/blob/4a7d0c69ab4316eebdee3e98320292386fe9a42d/pkg/util/token.go#L39-L60
func getFakeJWT(issuerURL string) (string, error) {
	fakeKey := []byte("fake-key")
	signingKey := jose.SigningKey{Algorithm: jose.HS256, Key: fakeKey}
	signingOptions := (&jose.SignerOptions{}).WithType("JWT")

	signer, err := jose.NewSigner(signingKey, signingOptions)
	if err != nil {
		return "", err
	}

	fakeClaims := jwt.Claims{
		Subject:   "fakeissuer",
		Issuer:    issuerURL,
		NotBefore: jwt.NewNumericDate(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)),
		Audience:  jwt.Audience(nil),
	}

	fakeJWT, err := jwt.Signed(signer).Claims(fakeClaims).CompactSerialize()
	if err != nil {
		return "", err
	}

	return fakeJWT, nil
}
