package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/claims"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/health"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/user"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
)

const (
	authorizationHeader              = "Authorization"
	impersonateUserHeader            = "Impersonate-User"
	impersonateGroupHeader           = "Impersonate-Group"
	impersonateUserExtraHeaderPrefix = "Impersonate-Extra-"
)

// ClientInterface ...
type ClientInterface interface {
	ReadinessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request)
	LivenessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request)
	AzadKubeProxyHandler(ctx context.Context, p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request)
	ErrorHandler(ctx context.Context) func(w http.ResponseWriter, r *http.Request, err error)
}

// Client ...
type Client struct {
	Config       config.Config
	CacheClient  cache.ClientInterface
	OIDCVerifier *oidc.IDTokenVerifier
	UserClient   user.ClientInterface
	ClaimsClient claims.ClientInterface
	HealthClient health.ClientInterface
}

// NewHandlersClient ...
func NewHandlersClient(ctx context.Context, config config.Config, cacheClient cache.ClientInterface, userClient user.ClientInterface, claimsClient claims.ClientInterface, healthClient health.ClientInterface) (ClientInterface, error) {
	oidcVerifier, err := claimsClient.GetOIDCVerifier(ctx, config.TenantID, config.ClientID)
	if err != nil {
		return nil, err
	}

	handlersClient := &Client{
		Config:       config,
		CacheClient:  cacheClient,
		OIDCVerifier: oidcVerifier,
		UserClient:   userClient,
		ClaimsClient: claimsClient,
		HealthClient: healthClient,
	}

	return handlersClient, nil
}

// ReadinessHandler ...
func (client *Client) ReadinessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)
	ready, err := client.HealthClient.Ready(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
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
func (client *Client) LivenessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\": \"ok\"}")); err != nil {
			log.Error(err, "Could not write response data")
		}
	}
}

// AzadKubeProxyHandler ...
func (client *Client) AzadKubeProxyHandler(ctx context.Context, p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		token, err := util.GetBearerToken(r)
		if err != nil {
			log.Error(err, "Unable to extract Bearer token")
			http.Error(w, "Unable to extract Bearer token", http.StatusForbidden)
			return
		}

		tokenHash := util.GetEncodedHash(token)

		// Verify user token
		verifiedToken, err := client.OIDCVerifier.Verify(ctx, token)
		if err != nil {
			log.Error(err, "Unable to verify token")
			http.Error(w, "Unable to verify token", http.StatusForbidden)
			return
		}

		// Use the token hash to get the user object from cache
		user, found, err := client.CacheClient.GetUser(ctx, tokenHash)
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
			claims, err := client.ClaimsClient.NewClaims(verifiedToken)
			if err != nil {
				log.Error(err, "Unable to get claims")
				http.Error(w, "Unable to get claims", http.StatusForbidden)
				return
			}

			// Get the user object
			user, err = client.UserClient.GetUser(ctx, claims.Username, claims.ObjectID)
			if err != nil {
				log.Error(err, "Unable to get user")
				http.Error(w, "Unable to get user", http.StatusForbidden)
				return
			}

			// Check if number of groups more than the configured limit
			if len(user.Groups) > client.Config.AzureADMaxGroupCount-1 {
				log.Error(errors.New("Max groups reached"), "The user is member of more groups than allowed to be passed to the Kubernetes API", "groupCount", len(user.Groups), "username", user.Username, "config.AzureADMaxGroupCount", client.Config.AzureADMaxGroupCount)
				http.Error(w, "Too many groups", http.StatusForbidden)
				return
			}

			err = client.CacheClient.SetUser(ctx, tokenHash, user)
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
		r.Header.Add(authorizationHeader, fmt.Sprintf("Bearer %s", client.Config.KubernetesConfig.Token))

		// Add the impersonation header for the users
		r.Header.Add(impersonateUserHeader, user.Username)

		// Add a new impersonation header per group
		for _, group := range user.Groups {
			switch client.Config.GroupIdentifier {
			case models.NameGroupIdentifier:
				r.Header.Add(impersonateGroupHeader, group.Name)
			case models.ObjectIDGroupIdentifier:
				r.Header.Add(impersonateGroupHeader, group.ObjectID)
			default:
				log.Error(errors.New("Unknown groups identifier"), "Unknown groups identifier", "GroupIdentifier", client.Config.GroupIdentifier)
				http.Error(w, "Unexpected error", http.StatusInternalServerError)
				return
			}
		}

		log.Info("Request", "path", r.URL.Path, "username", user.Username, "userType", user.Type, "groupCount", len(user.Groups), "cachedUser", found)

		p.ServeHTTP(w, r)
	}
}

// ErrorHandler ...
func (client *Client) ErrorHandler(ctx context.Context) func(w http.ResponseWriter, r *http.Request, err error) {
	log := logr.FromContext(ctx)

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
