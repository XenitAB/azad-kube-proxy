package reverseproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	"k8s.io/apiserver/pkg/authentication/request/bearertoken"
	"k8s.io/apiserver/plugin/pkg/authenticator/token/oidc"
)

// ReverseProxy returns common functions
type ReverseProxy struct {
	Authenticator *bearertoken.Authenticator
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
	log.Info("Initializing reverse proxy", "ListnerAddress", config.ListnerAddress)
	proxy := httputil.NewSingleHostReverseProxy(config.KubernetesConfig.URL)
	proxy.ErrorHandler = errorHandler(ctx)
	proxy.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	router := mux.NewRouter()
	router.HandleFunc("/readyz", readinessHandler(ctx)).Methods("GET")
	router.HandleFunc("/healthz", livenessHandler(ctx)).Methods("GET")

	log.Info("Waiting for OIDC to initialize", "tenantID", config.TenantID)
	auther, err := getAuthenticator(ctx, config)
	if err != nil {
		log.Error(err, "Failed to initialize OIDC", "tenantID", config.TenantID)
		return err
	}
	log.Info("OIDC initialized", "tenantID", config.TenantID)

	rp := &ReverseProxy{
		Authenticator: auther,
	}

	router.PathPrefix("/").HandlerFunc(proxyHandler(ctx, cache, proxy, config, rp))
	srv := &http.Server{Addr: config.ListnerAddress, Handler: router}

	// Start HTTP server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "Http Server Error")
		}
	}()
	log.Info("Server started")

	// Blocks until singal is sent
	<-done
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

func getAuthenticator(ctx context.Context, config config.Config) (*bearertoken.Authenticator, error) {
	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", config.TenantID)

	tokenAuther, err := oidc.New(oidc.Options{
		ClientID:             config.ClientID,
		GroupsClaim:          "groups",
		GroupsPrefix:         "",
		IssuerURL:            issuerURL,
		UsernameClaim:        "preferred_username",
		UsernamePrefix:       "",
		SupportedSigningAlgs: []string{"RS256"},
		CAFile:               "",
		RequiredClaims:       map[string]string{},
	})
	if err != nil {
		return nil, err
	}
	bearerToken := bearertoken.New(tokenAuther)

	fakeJWT, err := getFakeJWT(issuerURL)
	if err != nil {
		return nil, err
	}

	err = util.Retry(6, 5*time.Second, func() (err error) {
		fakeReq := &http.Request{
			RemoteAddr: "fakeRemoteAddress",
			Header:     http.Header{},
		}
		fakeReq.Header.Add("Authorization", fmt.Sprintf("Bearer %s", fakeJWT))
		_, _, err = bearerToken.AuthenticateRequest(fakeReq)
		if err != nil && !strings.HasSuffix(err.Error(), "authenticator not initialized") {
			return nil
		}

		return err
	})
	if err != nil {
		return nil, err
	}

	return bearerToken, nil
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
