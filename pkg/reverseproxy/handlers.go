package reverseproxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/patrickmn/go-cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/client-go/transport"
)

func readinessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\": \"ok\"}")); err != nil {
			log.Error(err, "Could not write response data")
		}
	}
}

func livenessHandler(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\": \"ok\"}")); err != nil {
			log.Error(err, "Could not write response data")
		}
	}
}

func proxyHandler(ctx context.Context, cache *cache.Cache, p *httputil.ReverseProxy, config config.Config, rp *ReverseProxy) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		reqAuthorizationHeaderSha256 := sha256.Sum256([]byte(r.Header.Get("Authorization")))
		reqAuthorizationHeaderHash := hex.EncodeToString(reqAuthorizationHeaderSha256[:])

		distributedGroupClaims := false
		userCacheKey := fmt.Sprintf("%s-username", reqAuthorizationHeaderHash)
		groupsCacheKey := fmt.Sprintf("%s-groups", reqAuthorizationHeaderHash)
		var info *authenticator.Response
		var username string
		var groups []string
		var found bool

		userCacheResponse, found := cache.Get(userCacheKey)

		if !found {
			// Validate user token
			var ok bool
			var err error

			info, ok, err = rp.Authenticator.AuthenticateRequest(r)
			if err != nil {
				if !strings.Contains(err.Error(), "could not expand distributed claims") {
					log.Error(err, "Unable to verify user token")
					http.Error(w, "Unable to verify user token", http.StatusForbidden)
					return
				}
				distributedGroupClaims = true
			}

			// TODO: Is the user really valid? Should change from "k8s.io/apiserver/pkg/authentication/authenticator" to something more generic
			if !ok {
				if !strings.Contains(err.Error(), "could not expand distributed claims") {
					log.Error(errors.New("User unauthorized"), "User unauthorized")
					http.Error(w, "User unauthorized", http.StatusForbidden)
					return
				}
			}

			// Validate that client isn't sending impersonation headers
			for h := range r.Header {
				if strings.ToLower(h) == strings.ToLower(transport.ImpersonateUserHeader) || strings.ToLower(h) == strings.ToLower(transport.ImpersonateGroupHeader) || strings.HasPrefix(strings.ToLower(h), strings.ToLower(transport.ImpersonateUserExtraHeaderPrefix)) {
					log.Error(errors.New("Client sending impersonation headers"), "Client sending impersonation headers")
					http.Error(w, "User unauthorized", http.StatusForbidden)
					return
				}
			}

			username = info.User.GetName()

			if distributedGroupClaims {
				groups = []string{"LOL"}
			}
			if !distributedGroupClaims {
				groups = info.User.GetGroups()
			}

			cache.Set(userCacheKey, username, 5*time.Minute)
			cache.Set(groupsCacheKey, groups, 5*time.Minute)
		}

		if found {
			username = userCacheResponse.(string)

			groupsCacheResponse, found := cache.Get(groupsCacheKey)
			if !found {
				log.Error(errors.New("Cache"), "Unable to find groups in cache", "groupsCacheKey", groupsCacheKey)
				http.Error(w, "Unable to find groups", http.StatusForbidden)
				return
			}

			groups = groupsCacheResponse.([]string)
		}

		// Remove the Authorization header that is sent to the server
		r.Header.Del("Authorization")

		// Add a new Authorization header with the token from the token path
		r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", config.KubernetesConfig.Token))

		// Add the impersonation header for the users
		r.Header.Add("Impersonate-User", username)

		// Create variable for the groups object

		// Logic if not using distributed claims
		groups = info.User.GetGroups()

		// Logic if using distributed claim
		if distributedGroupClaims {

		}

		// Add a new header per group
		groups = info.User.GetGroups()
		for _, group := range groups {
			if strings.Contains(group, config.AzureADGroupPrefix) {
				log.Info("Impersonate-Group", "GroupName", group)
				r.Header.Add("Impersonate-Group", group)
			}
		}

		log.Info("Request", "path", r.URL.Path)

		p.ServeHTTP(w, r)
	}
}

func errorHandler(ctx context.Context) func(w http.ResponseWriter, r *http.Request, err error) {
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
