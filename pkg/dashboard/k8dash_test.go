package dashboard

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

func TestK8dashPreAuth(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	req.Header.Add("Authorization", "Bearer fake-token")

	config := config.Config{
		TenantID: tenantID,
	}

	client, err := newK8dashClient(ctx, config)
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
	router.Handle("/", proxy)
	router.Use(client.preAuth)
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

	expectedCookieFound := false
	authorizationCookie := &http.Cookie{}
	for _, cookie := range rr.Result().Cookies() {
		if cookie.Name == "Authorization" {
			expectedCookieFound = true
			authorizationCookie = cookie
		}
	}

	if !expectedCookieFound {
		t.Error("expected cookie 'Authorization' not found in response")
	}

	if expectedCookieFound && authorizationCookie.Value != "fake-token" {
		t.Errorf("cookie 'Authorization' did not contain expected 'fake-token' but: %s", authorizationCookie.Value)
	}
}
