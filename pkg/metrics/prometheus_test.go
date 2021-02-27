package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
)

func TestPrometheusMetricsHandler(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	client := newPrometheusClient(ctx)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
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
	expectedStringContains := "# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles."
	if !strings.Contains(rr.Body.String(), expectedStringContains) {
		t.Errorf("handler returned unexpected body: got %v expected to contain %v",
			rr.Body.String(), expectedStringContains)
	}
}
