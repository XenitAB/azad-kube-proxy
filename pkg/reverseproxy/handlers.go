package reverseproxy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/go-logr/logr"

	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"k8s.io/apiserver/pkg/endpoints/request"
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

func proxyHandler(ctx context.Context, p *httputil.ReverseProxy, config config.Config, rp *ReverseProxy) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
		// Validate user token
		info, ok, err := rp.Authenticator.AuthenticateRequest(r)
		if err != nil {
			log.Error(err, "Unable to verify user token")
			http.Error(w, "Unable to verify user token", http.StatusForbidden)
			return
		}

		if !ok {
			log.Error(err, "User unauthorized")
			http.Error(w, "User unauthorized", http.StatusForbidden)
			return
		}

		// Validate that client isn't sending impersonation headers
		for h := range r.Header {
			if strings.ToLower(h) == strings.ToLower(transport.ImpersonateUserHeader) || strings.ToLower(h) == strings.ToLower(transport.ImpersonateGroupHeader) || strings.HasPrefix(strings.ToLower(h), strings.ToLower(transport.ImpersonateUserExtraHeaderPrefix)) {
				log.Error(errors.New("Client sending impersonation headers"), "Client sending impersonation headers")
				http.Error(w, "User unauthorized", http.StatusForbidden)
				return
			}
		}

		// Validate that the we are able to get a user
		user, ok := request.UserFrom(r.Context())
		if !ok || len(user.GetName()) == 0 {
			log.Error(errors.New("Unable to get user"), "Unable to get user", "user", user, "ok", ok)
			http.Error(w, "User unauthorized", http.StatusForbidden)
			return
		}

		r = r.WithContext(request.WithUser(r.Context(), info.User))

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
