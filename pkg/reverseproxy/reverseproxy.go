package reverseproxy

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

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	oidc "github.com/coreos/go-oidc"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// ReverseProxy returns common functions
type ReverseProxy struct {
	OIDCVerifier        *oidc.IDTokenVerifier
	AzureADGroupsClient graphrbac.GroupsClient
	AzureADUsersClient  graphrbac.UsersClient
}

// Start launches the reverse proxy
func Start(ctx context.Context, config config.Config) error {
	log := logr.FromContext(ctx)

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

	// Initiate memory cache
	cache := cache.New(5*time.Minute, 10*time.Minute)

	// Configure revers proxy and http server
	log.Info("Initializing reverse proxy", "ListenerAddress", config.ListenerAddress)
	proxy := httputil.NewSingleHostReverseProxy(config.KubernetesConfig.URL)
	proxy.ErrorHandler = errorHandler(ctx)
	proxy.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	router := mux.NewRouter()
	router.HandleFunc("/readyz", readinessHandler(ctx)).Methods("GET")
	router.HandleFunc("/healthz", livenessHandler(ctx)).Methods("GET")

	oidcVerifier, err := getOIDCVerifier(ctx, config)
	if err != nil {
		return err
	}

	groupsClient, err := azure.GetAzureADGroupsClient(ctx, config)
	if err != nil {
		return err
	}

	usersClient, err := azure.GetAzureADUsersClient(ctx, config)
	if err != nil {
		return err
	}

	rp := &ReverseProxy{
		OIDCVerifier:        oidcVerifier,
		AzureADGroupsClient: groupsClient,
		AzureADUsersClient:  usersClient,
	}

	// Initiate Azure AD group sync
	graphFilter := fmt.Sprintf("startswith(displayName,'%s')", config.AzureADGroupPrefix)
	if config.AzureADGroupPrefix == "" {
		graphFilter = ""
	}

	syncTicker, syncChan, err := azure.SyncTickerAzureADGroups(ctx, config, rp.AzureADGroupsClient, graphFilter, 5*time.Minute, cache)
	if err != nil {
		return err
	}

	router.PathPrefix("/").HandlerFunc(proxyHandler(ctx, cache, proxy, config, rp))
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
