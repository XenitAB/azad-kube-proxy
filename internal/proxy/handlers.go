package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/internal/cache"
	"github.com/xenitab/azad-kube-proxy/internal/config"
	"github.com/xenitab/azad-kube-proxy/internal/health"
	"github.com/xenitab/azad-kube-proxy/internal/models"
	"github.com/xenitab/azad-kube-proxy/internal/user"
	"github.com/xenitab/azad-kube-proxy/internal/util"
	"github.com/xenitab/go-oidc-middleware/options"
)

const (
	authorizationHeader              = "Authorization"
	impersonateUserHeader            = "Impersonate-User"
	impersonateGroupHeader           = "Impersonate-Group"
	impersonateUserExtraHeaderPrefix = "Impersonate-Extra-"
)

type Handler interface {
	ReadinessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request)
	LivenessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request)
	AzadKubeProxyHandler(ctx context.Context, p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request)
	ErrorHandler(ctx context.Context) func(w http.ResponseWriter, r *http.Request, err error)
}

type handler struct {
	CacheClient  cache.ClientInterface
	UserClient   user.ClientInterface
	HealthClient health.ClientInterface

	cfg             *config.Config
	groupIdentifier models.GroupIdentifier
	kubernetesToken string
}

func newHandlersClient(ctx context.Context, cfg *config.Config, cacheClient cache.ClientInterface, userClient user.ClientInterface, healthClient health.ClientInterface) (*handler, error) {
	groupIdentifier, err := models.GetGroupIdentifier(cfg.GroupIdentifier)
	if err != nil {
		return nil, err
	}

	kubernetesToken, err := util.GetStringFromFile(ctx, cfg.KubernetesAPITokenPath)
	if err != nil {
		return nil, err
	}

	handlersClient := &handler{
		CacheClient:     cacheClient,
		UserClient:      userClient,
		HealthClient:    healthClient,
		cfg:             cfg,
		groupIdentifier: groupIdentifier,
		kubernetesToken: kubernetesToken,
	}

	return handlersClient, nil
}

// ReadinessHandler ...
func (c *handler) ReadinessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContextOrDiscard(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		ready, err := c.HealthClient.Ready(ctx)
		if !ready {
			log.Error(err, "Ready check failed")
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte("{\"status\": \"error\"}")); err != nil {
				log.Error(err, "Could not write response data")
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\": \"ok\"}")); err != nil {
			log.Error(err, "Could not write response data")
		}
	}
}

// LivenessHandler ...
func (h *handler) LivenessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContextOrDiscard(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		live, err := h.HealthClient.Live(ctx)
		if !live {
			log.Error(err, "Live check failed")
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte("{\"status\": \"error\"}")); err != nil {
				log.Error(err, "Could not write response data")
			}
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\": \"ok\"}")); err != nil {
			log.Error(err, "Could not write response data")
		}
	}
}

// AzadKubeProxyHandler ...
func (h *handler) AzadKubeProxyHandler(ctx context.Context, p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContextOrDiscard(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		externalClaims, ok := r.Context().Value(options.DefaultClaimsContextKeyName).(externalAzureADClaims)
		if !ok {
			log.Error(fmt.Errorf("unable to typecast claims"), "not able to typecast claims to externalAzureADClaims")
			http.Error(w, "Unexpected error", http.StatusInternalServerError)
			return
		}

		claims, err := toInternalAzureADClaims(&externalClaims)
		if err != nil {
			log.Error(err, "not able to convert rawClaims to azureClaims")
			http.Error(w, "Unexpected error", http.StatusInternalServerError)
			return
		}

		// Use the token hash to get the user object from cache
		user, found, err := h.CacheClient.GetUser(ctx, claims.sub)
		if err != nil {
			log.Error(err, "Unable to get cached user object")
			http.Error(w, "Unexpected error", http.StatusInternalServerError)
			return
		}

		// Verify that client isn't sending impersonation headers
		for h := range r.Header {
			if strings.EqualFold(h, impersonateUserHeader) || strings.EqualFold(h, impersonateGroupHeader) || strings.HasPrefix(strings.ToLower(h), strings.ToLower(impersonateUserExtraHeaderPrefix)) {
				log.Error(errors.New("Client sending impersonation headers"), "Client sending impersonation headers")
				http.Error(w, "User unauthorized", http.StatusForbidden)
				return
			}
		}

		// Get the user from the token if no cache was found
		if !found {
			// Get the user object
			user, err = h.UserClient.GetUser(ctx, claims.username, claims.objectID)
			if err != nil {
				log.Error(err, "Unable to get user")
				http.Error(w, "Unable to get user", http.StatusForbidden)
				return
			}

			// Check if number of groups more than the configured limit
			if len(user.Groups) > h.cfg.AzureADMaxGroupCount-1 {
				log.Error(errors.New("max groups reached"), "the user is member of more groups than allowed to be passed to the Kubernetes API", "groupCount", len(user.Groups), "username", user.Username, "config.AzureADMaxGroupCount", h.cfg.AzureADMaxGroupCount)
				http.Error(w, "Too many groups", http.StatusForbidden)
				return
			}

			err = h.CacheClient.SetUser(ctx, claims.sub, user)
			if err != nil {
				log.Error(err, "Unable to set cache for user object")
				http.Error(w, "Unexpected error", http.StatusInternalServerError)
			}
		}

		// Remove the Authorization header that is sent to the server
		r.Header.Del(authorizationHeader)

		// Remove WebSocket Authorization header (base64url.bearer.authorization.k8s.io.<bearer>) that is sent to the server
		wsProtoString := util.StripWebSocketBearer(r.Header.Get("Sec-WebSocket-Protocol"))
		r.Header.Del("Sec-WebSocket-Protocol")
		r.Header.Add("Sec-WebSocket-Protocol", wsProtoString)

		// Add a new Authorization header with the token from the token path
		r.Header.Add(authorizationHeader, fmt.Sprintf("Bearer %s", h.kubernetesToken))

		// Add the impersonation header for the users
		r.Header.Add(impersonateUserHeader, user.Username)

		// Add a new impersonation header per group
		for _, group := range user.Groups {
			switch h.groupIdentifier {
			case models.NameGroupIdentifier:
				r.Header.Add(impersonateGroupHeader, group.Name)
			case models.ObjectIDGroupIdentifier:
				r.Header.Add(impersonateGroupHeader, group.ObjectID)
			default:
				log.Error(errors.New("unknown groups identifier"), "unknown groups identifier", "GroupIdentifier", h.cfg.GroupIdentifier)
				http.Error(w, "Unexpected error", http.StatusInternalServerError)
				return
			}
		}

		log.Info("Request", "path", r.URL.Path, "username", user.Username, "userType", user.Type, "groupCount", len(user.Groups), "cachedUser", found)

		incrementRequestCount(r)

		p.ServeHTTP(w, r)
	}
}

// ErrorHandler ...
func (h *handler) ErrorHandler(ctx context.Context) func(w http.ResponseWriter, r *http.Request, err error) {
	log := logr.FromContextOrDiscard(ctx)

	return func(w http.ResponseWriter, r *http.Request, err error) {
		if err == nil {
			log.Error(err, "error nil")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		switch err {
		default:
			log.Error(err, "Unexpected error")
			http.Error(w, "", http.StatusInternalServerError)
		}
	}
}
