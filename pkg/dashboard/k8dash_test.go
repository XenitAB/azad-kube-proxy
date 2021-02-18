package dashboard

import (
	"context"
	"fmt"
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

	cases := []struct {
		requestFunc            func() (*http.Request, string, error)
		expectedStringContains string
		expectedStatusCode     int
	}{
		{
			requestFunc: func() (*http.Request, string, error) {
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					return nil, "", fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				req.Header.Add("Authorization", "Bearer fake-token")
				return req, "fake-token", nil
			},
			expectedStringContains: "{\"fake\": true}",
			expectedStatusCode:     http.StatusOK,
		},
		{
			requestFunc: func() (*http.Request, string, error) {
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					return nil, "", fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				req.Header.Add("Authorization", "Bearer")
				return req, "", nil
			},
			expectedStringContains: "{\"fake\": true}",
			expectedStatusCode:     http.StatusOK,
		},
		{
			requestFunc: func() (*http.Request, string, error) {
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					return nil, "", fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				req.Header.Add("Authorization", "Bearer ")
				return req, "", nil
			},
			expectedStringContains: "{\"fake\": true}",
			expectedStatusCode:     http.StatusOK,
		},
		{
			requestFunc: func() (*http.Request, string, error) {
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					return nil, "", fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				return req, "", nil
			},
			expectedStringContains: "{\"fake\": true}",
			expectedStatusCode:     http.StatusOK,
		},
		{
			requestFunc: func() (*http.Request, string, error) {
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					return nil, "", fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				req.Header.Add("Authorization", "abc 123 fake")
				return req, "", nil
			},
			expectedStringContains: "{\"fake\": true}",
			expectedStatusCode:     http.StatusOK,
		},
		{
			requestFunc: func() (*http.Request, string, error) {
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					return nil, "", fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				req.Header.Add("Authorization", "Bearer token-fake")
				return req, "token-fake", nil
			},
			expectedStringContains: "{\"fake\": true}",
			expectedStatusCode:     http.StatusOK,
		},
	}

	for _, c := range cases {
		req, expectedCookieValue, err := c.requestFunc()
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
		if rr.Code != c.expectedStatusCode {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, c.expectedStatusCode)
		}

		if !strings.Contains(rr.Body.String(), c.expectedStringContains) {
			t.Errorf("handler returned unexpected body: got %v expected to contain %v",
				rr.Body.String(), c.expectedStringContains)
		}

		if expectedCookieValue != "" {
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

			if expectedCookieFound && authorizationCookie.Value != expectedCookieValue {
				t.Errorf("cookie 'Authorization' did not contain expected '%s' but: %s", expectedCookieValue, authorizationCookie.Value)
			}
		}

		if expectedCookieValue == "" {
			expectedCookieFound := false
			for _, cookie := range rr.Result().Cookies() {
				if cookie.Name == "Authorization" {
					expectedCookieFound = true
				}
			}

			if expectedCookieFound {
				t.Error("expected cookie 'Authorization' was not expected to be found in response")
			}
		}
	}
}

func TestK8dashGetOIDC(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	cases := []struct {
		requestFunc            func() (*http.Request, error)
		config                 config.Config
		expectedStringContains string
		expectedStatusCode     int
	}{
		{
			requestFunc: func() (*http.Request, error) {
				req, err := http.NewRequest("GET", "/oidc", nil)
				if err != nil {
					return nil, fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				return req, nil
			},
			config: config.Config{
				TenantID: tenantID,
				K8dashConfig: config.K8dashConfig{
					ClientID: "00000000-0000-0000-0000-000000000000",
					Scope:    "fake-scope",
				},
			},
			expectedStringContains: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantID),
			expectedStatusCode:     http.StatusOK,
		},
		{
			requestFunc: func() (*http.Request, error) {
				req, err := http.NewRequest("GET", "/oidc", nil)
				if err != nil {
					return nil, fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				return req, nil
			},
			config: config.Config{
				TenantID: tenantID,
				K8dashConfig: config.K8dashConfig{
					ClientID: "00000000-0000-0000-0000-000000000000",
					Scope:    "fake-scope",
				},
			},
			expectedStringContains: "client_id=00000000-0000-0000-0000-000000000000",
			expectedStatusCode:     http.StatusOK,
		},
		{
			requestFunc: func() (*http.Request, error) {
				req, err := http.NewRequest("GET", "/oidc", nil)
				if err != nil {
					return nil, fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				return req, nil
			},
			config: config.Config{
				TenantID: tenantID,
				K8dashConfig: config.K8dashConfig{
					ClientID: "00000000-0000-0000-0000-000000000000",
					Scope:    "fake-scope",
				},
			},
			expectedStringContains: "scope=fake-scope",
			expectedStatusCode:     http.StatusOK,
		},
	}

	for _, c := range cases {
		client, err := newK8dashClient(ctx, c.config)
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		req, err := c.requestFunc()
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/oidc", client.getOIDC(ctx)).Methods("GET")
		router.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if rr.Code != c.expectedStatusCode {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, c.expectedStatusCode)
		}

		if !strings.Contains(rr.Body.String(), c.expectedStringContains) {
			t.Errorf("handler returned unexpected body: got %v expected to contain %v",
				rr.Body.String(), c.expectedStringContains)
		}
	}
}

func TestK8dashPostOIDC(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	cases := []struct {
		requestFunc            func() (*http.Request, error)
		config                 config.Config
		expectedStringContains string
		expectedStatusCode     int
	}{
		{
			requestFunc: func() (*http.Request, error) {
				req, err := http.NewRequest("POST", "/oidc", nil)
				if err != nil {
					return nil, fmt.Errorf("Expected err to be nil but it was %q", err)
				}
				return req, nil
			},
			config: config.Config{
				TenantID: tenantID,
				K8dashConfig: config.K8dashConfig{
					ClientID:     "00000000-0000-0000-0000-000000000000",
					ClientSecret: "fake-secret",
					Scope:        "fake-scope",
				},
			},
			expectedStringContains: "",
			expectedStatusCode:     http.StatusInternalServerError,
		},
	}

	for _, c := range cases {
		client, err := newK8dashClient(ctx, c.config)
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		req, err := c.requestFunc()
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/oidc", client.postOIDC(ctx)).Methods("POST")
		router.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if rr.Code != c.expectedStatusCode {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, c.expectedStatusCode)
		}

		if c.expectedStringContains != "" {
			if !strings.Contains(rr.Body.String(), c.expectedStringContains) {
				t.Errorf("handler returned unexpected body: got %v expected to contain %v",
					rr.Body.String(), c.expectedStringContains)
			}
		}
	}
}
