package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
)

func TestNoneMetricsHandler(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	client := newNoneClient(ctx)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	fakeBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{\"fake\": true}"))
	}))
	defer fakeBackend.Close()
	fakeBackendURL, err := url.Parse(fakeBackend.URL)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(fakeBackendURL)
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.PathPrefix("/").Handler(proxy)
	router, err = client.MetricsHandler(ctx, router)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	router.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check the response body is what we expect.
	expectedStringContains := "{\"fake\": true}"
	if !strings.Contains(rr.Body.String(), expectedStringContains) {
		t.Errorf("handler returned unexpected body: got %v expected to contain %v",
			rr.Body.String(), expectedStringContains)
	}
}
