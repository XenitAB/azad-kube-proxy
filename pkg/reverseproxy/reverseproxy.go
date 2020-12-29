package reverseproxy

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"

	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

// Start launches the reverse proxy
func Start(ctx context.Context, config config.Config) error {
	log := logr.FromContext(ctx)

	// Signal handler
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	insecureSkipVerify := false
	if config.ValidateKubernetesCertificate == false {
		insecureSkipVerify = true
	}
	log.Info("Should we verify the cert?", "insecureSkipVerify", insecureSkipVerify, "config.ValidateKubernetesCertificate", config.ValidateKubernetesCertificate)

	// Configure revers proxy and http server
	log.Info("Starting reverse proxy", "ListnerAddress", config.ListnerAddress)
	proxy := httputil.NewSingleHostReverseProxy(config.KubernetesAPIUrl)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureSkipVerify},
	}
	router := mux.NewRouter()
	router.HandleFunc("/readyz", readinessHandler(ctx)).Methods("GET")
	router.HandleFunc("/healthz", livenessHandler(ctx)).Methods("GET")
	router.PathPrefix("/").HandlerFunc(proxyHandler(ctx, proxy, config))
	srv := &http.Server{Addr: config.ListnerAddress, Handler: router}

	// Start HTTP server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err, "Http Server Error")
		}
	}()
	log.Info("Server started")

	// Blocks until singal is sent
	<-done
	log.Info("Server stopped")

	// Shutdown http server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error(err, "Server shutdown failed")
		return err
	}

	log.Info("Server exited properly")
	return nil
}

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
