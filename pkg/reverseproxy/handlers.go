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
	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/claims"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

const (
	impersonateUserHeader            = "Impersonate-User"
	impersonateGroupHeader           = "Impersonate-Group"
	impersonateUserExtraHeaderPrefix = "Impersonate-Extra-"
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
		token := strings.Split(r.Header.Get("Authorization"), "Bearer ")[1]
		tokenHashSha256 := sha256.Sum256([]byte(token))
		tokenHash := hex.EncodeToString(tokenHashSha256[:])

		distributedGroupClaims := false
		userCacheKey := fmt.Sprintf("%s-username", tokenHash)
		groupsCacheKey := fmt.Sprintf("%s-groups", tokenHash)
		var username string
		var groups []string
		var found bool

		userCacheResponse, found := cache.Get(userCacheKey)

		if !found {
			// Validate user token
			verifiedToken, err := rp.Verifier.Verify(r.Context(), token)
			if err != nil {
				log.Error(err, "Unable to verify token")
				http.Error(w, "Unable to verify token", http.StatusForbidden)
				return
			}

			var tokenClaims claims.AzureClaims

			if err := verifiedToken.Claims(&tokenClaims); err != nil {
				log.Error(err, "Unable to get token claims")
				http.Error(w, "Unable to get token claims", http.StatusForbidden)
				return

			}

			// Validate that client isn't sending impersonation headers
			for h := range r.Header {
				if strings.ToLower(h) == strings.ToLower(impersonateUserHeader) || strings.ToLower(h) == strings.ToLower(impersonateGroupHeader) || strings.HasPrefix(strings.ToLower(h), strings.ToLower(impersonateUserExtraHeaderPrefix)) {
					log.Error(errors.New("Client sending impersonation headers"), "Client sending impersonation headers")
					http.Error(w, "User unauthorized", http.StatusForbidden)
					return
				}
			}

			username = tokenClaims.Username

			// TODO: Update tokenClaims.ClaimNames.Groups == "src1" to be more specific if there's another src1?
			if len(tokenClaims.Groups) == 0 && tokenClaims.ClaimNames.Groups != "" {
				distributedGroupClaims = true
			}

			if distributedGroupClaims {
				groups, err = azure.GetAzureADGroups(ctx, tokenClaims, config)
				if err != nil {
					log.Error(err, "Unable to get distributed claims")
					http.Error(w, "Unable to get distributed claims", http.StatusForbidden)
					return
				}
			}
			if !distributedGroupClaims {
				groups = tokenClaims.Groups
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

		// Add a new header per group
		for _, group := range groups {
			if strings.Contains(group, config.AzureADGroupPrefix) {
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
