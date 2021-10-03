package dashboard

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"golang.org/x/oauth2"
)

//go:embed static/k8dash/build/*
var assets embed.FS

// k8dashFS implements fs.FS
type k8dashFS struct {
	content embed.FS
}

func (c k8dashFS) Open(name string) (fs.File, error) {
	return c.content.Open(path.Join("static/k8dash/build/", name))
}

// K8dashClient ...
type k8dashClient struct {
	oidcProvider *oidc.Provider
	config       config.Config
	authClient   authInterface
}

func newK8dashClient(ctx context.Context, config config.Config) (k8dashClient, error) {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("Using dashboard: k8dash")

	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", config.TenantID)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return k8dashClient{}, err
	}

	clientID := config.K8dashConfig.ClientID
	clientSecret := config.K8dashConfig.ClientSecret
	scope := config.K8dashConfig.Scope

	return k8dashClient{
		oidcProvider: provider,
		config:       config,
		authClient:   newAuthClient(clientID, clientSecret, provider.Endpoint(), []string{scope}),
	}, nil
}

// DashboardHandler ...
func (client *k8dashClient) DashboardHandler(ctx context.Context, router *mux.Router) (*mux.Router, error) {
	log := logr.FromContextOrDiscard(ctx)

	assetManifest, err := fs.ReadFile(k8dashFS{assets}, "asset-manifest.json")
	if err != nil {
		log.Error(err, "Unable to open asset manifest")
		return nil, err
	}

	manifest := struct {
		Files     map[string]string
		Endpoints []string
	}{}

	err = json.Unmarshal(assetManifest, &manifest)
	if err != nil {
		log.Error(err, "Unable to unmarshal asset manifest")
		return nil, err
	}

	fsServer := http.FileServer(http.FS(k8dashFS{assets}))

	staticFiles := []string{
		"/favicon.ico",
		"/logo.png",
		"/manifest.json",
	}

	for _, v := range manifest.Files {
		path := strings.TrimPrefix(v, ".")
		staticFiles = append(staticFiles, path)
	}

	for _, file := range staticFiles {
		router.Path(file).Handler(fsServer).Methods("GET")
	}
	router.Path("/").Handler(fsServer).Methods("GET")
	router.HandleFunc("/oidc", client.getOIDC(ctx)).Methods("GET")
	router.HandleFunc("/oidc", client.postOIDC(ctx)).Methods("POST")
	router.Use(client.preAuth)

	return router, nil
}

func (client *k8dashClient) preAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.Contains(authHeader, "Bearer ") {
			next.ServeHTTP(w, r)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if r.URL.Path == "/" && token != "" {
			cookie := &http.Cookie{Name: "Authorization", Value: token, HttpOnly: false, MaxAge: 60}
			http.SetCookie(w, cookie)
		}

		next.ServeHTTP(w, r)
	})
}

func (client *k8dashClient) getOIDC(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContextOrDiscard(ctx)

	clientID := client.config.K8dashConfig.ClientID
	scope := client.config.K8dashConfig.Scope

	return func(w http.ResponseWriter, r *http.Request) {
		authURL, err := url.Parse(client.oidcProvider.Endpoint().AuthURL)
		if err != nil {
			log.Error(err, "Unable parse auth url")
			http.Error(w, "Unable parse auth url", http.StatusInternalServerError)
			return
		}

		query := authURL.Query()

		query.Set("client_id", clientID)
		query.Set("scope", scope)
		query.Set("response_type", "code")
		query.Set("response_mode", "query")

		authURLString := fmt.Sprintf("%s?%s", authURL.String(), query.Encode())

		body := struct {
			AuthorizationEndpoint string `json:"authEndpoint"`
		}{
			AuthorizationEndpoint: authURLString,
		}

		resBody, err := json.Marshal(&body)
		if err != nil {
			log.Error(err, "Unable to marshal response")
			http.Error(w, "Unable to marshal response", http.StatusInternalServerError)
			return
		}

		_, err = w.Write(resBody)
		if err != nil {
			log.Error(err, "Unable to get oidc authorization url")
			http.Error(w, "Unable to get oidc authorization url", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
	}
}

func (client *k8dashClient) postOIDC(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContextOrDiscard(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Code        string `json:"code"`
			RedirectURI string `json:"redirectUri"`
		}

		err := json.NewDecoder(r.Body).Decode(&reqBody)
		if err != nil {
			log.Error(err, "Unable to unmarshal request body")
			http.Error(w, "Unable to unmarshal request body", http.StatusInternalServerError)
			return
		}

		if reqBody.Code == "" || reqBody.RedirectURI == "" {
			log.Error(err, "Invalid request body", "Code", reqBody.Code, "RedirectURI", reqBody.RedirectURI)
			http.Error(w, "Invalid request body", http.StatusInternalServerError)
			return
		}

		oauth2Token, err := client.authClient.Exchange(ctx, reqBody.Code, reqBody.RedirectURI)
		if err != nil {
			log.Error(err, "Unable to get access token")
			http.Error(w, "Unable to get access token", http.StatusInternalServerError)
			return
		}

		body := struct {
			AccessToken string `json:"token"`
		}{
			AccessToken: oauth2Token.AccessToken,
		}

		resBody, err := json.Marshal(&body)
		if err != nil {
			log.Error(err, "Unable to marshal response")
			http.Error(w, "Unable to marshal response", http.StatusInternalServerError)
			return
		}

		_, err = w.Write(resBody)
		if err != nil {
			log.Error(err, "Unable to get oidc authorization url")
			http.Error(w, "Unable to get oidc authorization url", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
	}
}

type authInterface interface {
	Exchange(ctx context.Context, code, redirectURL string) (*oauth2.Token, error)
}

type authClient struct {
	oauth2Config *oauth2.Config
}

func newAuthClient(clientID, clientSecret string, endpoint oauth2.Endpoint, scopes []string) authInterface {
	return &authClient{
		oauth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     endpoint,
			Scopes:       scopes,
		},
	}
}

func (client *authClient) Exchange(ctx context.Context, code, redirectURL string) (*oauth2.Token, error) {
	oauth2Config := client.oauth2Config
	oauth2Config.RedirectURL = redirectURL
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	return oauth2Token, err
}
