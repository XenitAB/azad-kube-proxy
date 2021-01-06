package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/pkg/claims"
	"github.com/xenitab/azad-kube-proxy/pkg/util"
)

const (
	authorizationHeader              = "Authorization"
	impersonateUserHeader            = "Impersonate-User"
	impersonateGroupHeader           = "Impersonate-Group"
	impersonateUserExtraHeaderPrefix = "Impersonate-Extra-"
)

func (rp *Proxy) readinessHandler() func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(rp.Context)

	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\": \"ok\"}")); err != nil {
			log.Error(err, "Could not write response data")
		}
	}
}

func (rp *Proxy) livenessHandler() func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(rp.Context)

	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\": \"ok\"}")); err != nil {
			log.Error(err, "Could not write response data")
		}
	}
}

func (rp *Proxy) proxyHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(rp.Context)

	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		token, err := util.GetBearerToken(r)
		if err != nil {
			log.Error(err, "Unable to extract Bearer token")
			http.Error(w, "Unable to extract Bearer token", http.StatusForbidden)
			return
		}

		tokenHash := util.GetEncodedHash(token)

		// Use the token hash to get the user object from cache
		u, found, err := rp.Cache.GetUser(tokenHash)
		if err != nil {
			log.Error(err, "Unable to get cached user object")
			http.Error(w, "Unexpected error", http.StatusInternalServerError)
			return
		}

		// Get the user from the token if no cache was found
		if !found {
			// Verify user token
			verifiedToken, err := rp.OIDCVerifier.Verify(r.Context(), token)
			if err != nil {
				log.Error(err, "Unable to verify token")
				http.Error(w, "Unable to verify token", http.StatusForbidden)
				return
			}

			// Verify that client isn't sending impersonation headers
			for h := range r.Header {
				if strings.ToLower(h) == strings.ToLower(impersonateUserHeader) || strings.ToLower(h) == strings.ToLower(impersonateGroupHeader) || strings.HasPrefix(strings.ToLower(h), strings.ToLower(impersonateUserExtraHeaderPrefix)) {
					log.Error(errors.New("Client sending impersonation headers"), "Client sending impersonation headers")
					http.Error(w, "User unauthorized", http.StatusForbidden)
					return
				}
			}

			c, err := claims.NewClaims(verifiedToken)
			if err != nil {
				log.Error(err, "Unable to get claims")
				http.Error(w, "Unable to get claims", http.StatusForbidden)
				return
			}

			// Get the user object
			u, err = rp.UserClient.GetUser(c.Username, c.ObjectID, c.Groups)
			if err != nil {
				log.Error(err, "Unable to get user")
				http.Error(w, "Unable to get user", http.StatusForbidden)
				return
			}

			// Check if number of groups more than the configured limit
			if len(u.Groups) > rp.Config.AzureADMaxGroupCount {
				log.Error(errors.New("Max groups reached"), "The user is member of more groups than allowed to be passed to the Kubernetes API", "groupCount", len(u.Groups), "username", u.Username, "config.AzureADMaxGroupCount", rp.Config.AzureADMaxGroupCount)
				http.Error(w, "Too many groups", http.StatusForbidden)
				return
			}

			rp.Cache.SetUser(tokenHash, u)
		}

		// Remove the Authorization header that is sent to the server
		r.Header.Del(authorizationHeader)

		// Add a new Authorization header with the token from the token path
		r.Header.Add(authorizationHeader, fmt.Sprintf("Bearer %s", rp.Config.KubernetesConfig.Token))

		// Add the impersonation header for the users
		r.Header.Add(impersonateUserHeader, u.Username)

		// Add a new header per group
		for _, group := range u.Groups {
			r.Header.Add(impersonateGroupHeader, group.Name)
		}

		log.Info("Request", "path", r.URL.Path, "username", u.Username, "groupCount", len(u.Groups), "cachedUser", found)

		p.ServeHTTP(w, r)
	}
}

func (rp *Proxy) errorHandler(ctx context.Context) func(w http.ResponseWriter, r *http.Request, err error) {
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
