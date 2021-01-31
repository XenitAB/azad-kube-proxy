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

// K8sdashClient ...
type k8sdashClient struct {
	oidcProvider *oidc.Provider
	config       config.Config
}

func newK8sdashClient(ctx context.Context, config config.Config) (k8sdashClient, error) {
	log := logr.FromContext(ctx)
	log.Info("Using dashboard: k8sdash")

	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", config.TenantID)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return k8sdashClient{}, err
	}

	return k8sdashClient{
		oidcProvider: provider,
		config:       config,
	}, nil
}

// DashboardHandler ...
func (client *k8sdashClient) DashboardHandler(ctx context.Context, router *mux.Router) (*mux.Router, error) {
	log := logr.FromContext(ctx)

	k8sdashPath := os.Getenv("K8S_DASH_PATH")
	if k8sdashPath == "" {
		err := fmt.Errorf("K8S_DASH_PATH environment variable not set")
		log.Error(err, "")
		return nil, err
	}

	assetManifest, err := util.GetStringFromFile(ctx, fmt.Sprintf("%s/asset-manifest.json", k8sdashPath))
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

	fs := http.FileServer(http.Dir(k8sdashPath))

	for _, v := range manifest.Files {
		path := strings.TrimPrefix(v, ".")
		log.Info("Debug", "path", path)
		router.Path(path).Handler(fs)
	}

	static := []string{
		"/favicon.ico",
		"/logo.png",
		"/manifest.json",
	}

	for _, file := range static {
		router.Path(file).Handler(fs)
	}

	router.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, fmt.Sprintf("%s/index.html", k8sdashPath))
	}).Methods("GET")

	router.HandleFunc("/oidc", client.getOIDC(ctx)).Methods("GET")
	router.HandleFunc("/oidc", client.postOIDC(ctx)).Methods("POST")
	router.HandleFunc("/", client.postOIDC(ctx)).Methods("POST")

	return router, nil
}

func (client *k8sdashClient) getOIDC(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		authURL, err := url.Parse(client.oidcProvider.Endpoint().AuthURL)
		if err != nil {
			log.Error(err, "Unable pars auth url")
			http.Error(w, "Unable pars auth url", http.StatusInternalServerError)
			return
		}

		query := authURL.Query()

		query.Set("client_id", "0622715d-3443-4ca1-940e-0d2a360344a6")
		query.Set("scope", "https://k8s-api.azadkubeproxy.onmicrosoft.com/.default")
		query.Set("response_type", "code")

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

func (client *k8sdashClient) postOIDC(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

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

		oauth2Config := oauth2.Config{
			ClientID:     "0622715d-3443-4ca1-940e-0d2a360344a6",
			ClientSecret: "somethingsecret",
			RedirectURL:  reqBody.RedirectURI,
			Endpoint:     client.oidcProvider.Endpoint(),
			Scopes:       []string{"https://k8s-api.azadkubeproxy.onmicrosoft.com/.default"},
		}

		oauth2Token, err := oauth2Config.Exchange(ctx, reqBody.Code)
		if err != nil {
			log.Error(err, "Unable to get access token")
			http.Error(w, "Unable to get access token", http.StatusInternalServerError)
			return
		}

		accessToken := oauth2Token.AccessToken

		log.Info("Debug access token", "accessToken", accessToken)

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

// app.use('/', preAuth, express.static('public'));
// app.get('/oidc', getOidc);
// app.post('/oidc', postOidc);
// app.use('/*', createProxyMiddleware(proxySettings));
// app.use(handleErrors);

// const port = process.env.SERVER_PORT || 4654;
// http.createServer(app).listen(port);
// console.log(`Server started. Listening on port ${port}`);

// function preAuth(req, res, next) {
//     const auth = req.header('Authorization');

//     // If the request already contains an authorization header, pass it through to the client (as a cookie)
//     if (auth && req.method === 'GET' && req.path === '/') {
//         const value = auth.replace('Bearer ', '');
//         res.cookie('Authorization', value, {maxAge: 60, httpOnly: false});
//         console.log('Authorization header found. Passing through to client.');
//     }

//     next();
// }

// async function getOidc(req, res) {
//     try {
//         const authEndpoint = await getOidcEndpoint();
//         res.json({authEndpoint});
//     } catch (err) {
//         next(err);
//     }
// }

// async function postOidc(req, res, next) {
//     try {
//         const body = await toString(req);
//         const {code, redirectUri} = JSON.parse(body);
//         const token = await oidcAuthenticate(code, redirectUri);
//         res.json({token});
//     } catch (err) {
//         next(err);
//     }
// }
