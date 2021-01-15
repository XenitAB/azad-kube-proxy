package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	logrTesting "github.com/go-logr/logr/testing"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

func TestReadinessHandler(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	req, err := http.NewRequest("GET", "/readyz", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	server, err := NewProxyServer(ctx, config.Config{})
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/readyz", server.readinessHandler(ctx)).Methods("GET")
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `{"status": "ok"}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestLivenessHandler(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	server, err := NewProxyServer(ctx, config.Config{})
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/healthz", server.livenessHandler(ctx)).Methods("GET")
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `{"status": "ok"}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestAzadKubeProxyHandler(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	server, err := NewProxyServer(ctx, config.Config{})
	proxy := server.getReverseProxy(ctx)
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.PathPrefix("/").HandlerFunc(server.azadKubeProxyHandler(ctx, proxy))
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `{"status": "ok"}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
