package reverseproxy

import (
	"context"
	"net/http"
	"net/http/httputil"

	"github.com/go-logr/logr"

	"github.com/xenitab/azad-kube-proxy/pkg/config"
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

func proxyHandler(ctx context.Context, p *httputil.ReverseProxy, config config.Config) func(http.ResponseWriter, *http.Request) {
	log := logr.FromContext(ctx)

	return func(w http.ResponseWriter, r *http.Request) {
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
