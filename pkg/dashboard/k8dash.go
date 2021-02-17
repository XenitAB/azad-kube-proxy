package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
	"golang.org/x/oauth2"
)

// K8dashClient ...
type k8dashClient struct {
	oidcProvider *oidc.Provider
	config       config.Config
}

func newK8dashClient(ctx context.Context, config config.Config) (k8dashClient, error) {
	log := logr.FromContext(ctx)
	log.Info("Using dashboard: k8dash")

	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", config.TenantID)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return k8dashClient{}, err
	}

	return k8dashClient{
		oidcProvider: provider,
		config:       config,
	}, nil
}

// DashboardHandler ...
func (client *k8dashClient) DashboardHandler(ctx context.Context, router *mux.Router) (*mux.Router, error) {
	log := logr.FromContext(ctx)

	k8dashPath := os.Getenv("K8DASH_PATH")
	if k8dashPath == "" {
		err := fmt.Errorf("K8DASH_PATH environment variable not set")
		log.Error(err, "")
		return nil, err
	}

	assetManifest, err := util.GetStringFromFile(ctx, fmt.Sprintf("%s/asset-manifest.json", k8dashPath))
	if err != nil {
		log.Error(err, "Unable to open asset manifest")
		return nil, err
	}

	manifest := struct {
		Files     map[string]string
		Endpoints []string
	}{}

	err = json.Unmarshal([]byte(assetManifest), &manifest)
	if err != nil {
		log.Error(err, "Unable to unmarshal asset manifest")
		return nil, err
	}

	fs := http.FileServer(http.Dir(k8dashPath))

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
		router.Path(file).Handler(fs).Methods("GET")
	}

	router.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, fmt.Sprintf("%s/index.html", k8dashPath))
	}).Methods("GET")

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
		return
	})
}

func (client *k8dashClient) getOIDC(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

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
	log := logr.FromContext(ctx)

	clientID := client.config.K8dashConfig.ClientID
	clientSecret := client.config.K8dashConfig.ClientSecret
	scope := client.config.K8dashConfig.Scope

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
			return
		}

		oauth2Config := oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  reqBody.RedirectURI,
			Endpoint:     client.oidcProvider.Endpoint(),
			Scopes:       []string{scope},
		}

		oauth2Token, err := oauth2Config.Exchange(ctx, reqBody.Code)
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
