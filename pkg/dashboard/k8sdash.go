package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
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
func (client *k8sdashClient) DashboardHandler(ctx context.Context, router *mux.Router) *mux.Router {
	router.HandleFunc("/oidc", client.getOIDC(ctx)).Methods("GET")
	router.HandleFunc("/oidc", client.postOIDC(ctx)).Methods("POST")
	return router
}

func (client *k8sdashClient) getOIDC(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		body := struct {
			AuthorizationEndpoint string `json:"authEndpoint"`
		}{
			AuthorizationEndpoint: client.oidcProvider.Endpoint().AuthURL,
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
			ClientID:     client.config.ClientID,
			ClientSecret: client.config.ClientSecret,
			RedirectURL:  "https://k8s-api.azadkubeproxy.onmicrosoft.com",
			Endpoint:     client.oidcProvider.Endpoint(),
			Scopes:       []string{"https://k8s-api.azadkubeproxy.onmicrosoft.com/.default"},
		}

		oauth2Token, err := oauth2Config.Exchange(ctx, reqBody.Code)

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
