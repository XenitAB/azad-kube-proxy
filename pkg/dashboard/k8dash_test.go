package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

func TestNewK8sdashClient(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	cfg := config.Config{
		TenantID: tenantID,
	}

	_, err := newK8dashClient(ctx, cfg)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
}

func TestK8dashDashboardHandler(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")

	restoreK8dashPath := tempChangeEnv("K8DASH_PATH", "../../tmp/k8dash/build/")
	defer restoreK8dashPath()

	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	config := config.Config{
		TenantID: tenantID,
	}

	client, err := newK8dashClient(ctx, config)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	routerWithDashboard, err := client.DashboardHandler(ctx, router)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	routerWithDashboard.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// Check the response body is what we expect.
	expectedStringContains := "<!doctype html><html lang=\"en\"><head><script>let basePath"
	if !strings.Contains(rr.Body.String(), expectedStringContains) {
		t.Errorf("handler returned unexpected body: got %v expected to contain %v",
			rr.Body.String(), expectedStringContains)
	}
}

func tempChangeEnv(key, value string) func() {
	oldEnv := os.Getenv(key)
	os.Setenv(key, value)
	return func() { os.Setenv(key, oldEnv) }
}
