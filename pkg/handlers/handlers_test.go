package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	logrTesting "github.com/go-logr/logr/testing"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/user"
)

func TestReadinessHandler(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	req, err := http.NewRequest("GET", "/readyz", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	fakeCache := newFakeCache()
	proxyHandlers, err := NewHandlersClient(ctx, config.Config{}, fakeCache, &user.Client{})
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/readyz", proxyHandlers.ReadinessHandler(ctx)).Methods("GET")
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

	fakeCache := newFakeCache()
	proxyHandlers, err := NewHandlersClient(ctx, config.Config{}, fakeCache, &user.Client{})
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/healthz", proxyHandlers.LivenessHandler(ctx)).Methods("GET")
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
	clientID := getEnvOrSkip(t, "CLIENT_ID")
	clientSecret := getEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	spClientID := getEnvOrSkip(t, "TEST_USER_SP_CLIENT_ID")
	spClientSecret := getEnvOrSkip(t, "TEST_USER_SP_CLIENT_SECRET")
	spResource := getEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	token, err := getAccessToken(ctx, tenantID, spClientID, spClientSecret, fmt.Sprintf("%s/.default", spResource))
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	memCache, err := cache.NewMemoryCache(5*time.Minute, 10*time.Minute)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	fakeCache := newFakeCache()

	cases := []struct {
		request             *http.Request
		cacheClient         cache.ClientInterface
		expectedResCode     int
		expectedErrContains string
	}{
		{
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {fmt.Sprintf("Bearer %s", token.Token)},
				},
			},
			cacheClient:         memCache,
			expectedResCode:     http.StatusOK,
			expectedErrContains: "",
		},
		{
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {"Bearer"},
				},
			},
			cacheClient:         memCache,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "Unable to extract Bearer token",
		},
		{
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {"Bearer fake-token"},
				},
			},
			cacheClient:         memCache,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "Unable to verify token",
		},
		{
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {"Bearer fake-token"},
				},
			},
			cacheClient:         fakeCache,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "",
		},
	}

	fakeBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"fake\": true}"))
	}))
	defer fakeBackend.Close()
	fakeBackendURL, err := url.Parse(fakeBackend.URL)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	config := config.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
		CacheEngine:  models.MemoryCacheEngine,
		KubernetesConfig: config.KubernetesConfig{
			URL:   fakeBackendURL,
			Token: "fake-token",
		},
	}

	for _, c := range cases {
		proxyHandlers, err := NewHandlersClient(ctx, config, c.cacheClient, &user.Client{})
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		proxy := httputil.NewSingleHostReverseProxy(config.KubernetesConfig.URL)
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.PathPrefix("/").HandlerFunc(proxyHandlers.AzadKubeProxyHandler(ctx, proxy))
		router.ServeHTTP(rr, c.request)

		if rr.Code != c.expectedResCode {
			t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, c.expectedResCode)
		}

		expected := `{"fake": true}`
		if rr.Body.String() != expected && c.expectedErrContains == "" {
			t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
		}

		if c.expectedErrContains != "" {
			if !strings.Contains(rr.Body.String(), c.expectedErrContains) {
				t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), c.expectedErrContains)
			}
		}
	}
}

type fakeCache struct {
	fakeError error
	fakeFound bool
	fakeUser  models.User
	fakeGroup models.Group
}

func newFakeCache() *fakeCache {
	return &fakeCache{
		fakeError: nil,
		fakeFound: true,
		fakeUser: models.User{
			Username: "username",
			ObjectID: "00000000-0000-0000-0000-000000000000",
			Groups: []models.Group{
				{Name: "group1"},
			},
		},
		fakeGroup: models.Group{
			Name: "group1",
		},
	}
}

// GetUser ...
func (c *fakeCache) GetUser(ctx context.Context, s string) (models.User, bool, error) {
	return c.fakeUser, c.fakeFound, c.fakeError
}

// SetUser ...
func (c *fakeCache) SetUser(ctx context.Context, s string, u models.User) error {
	return c.fakeError
}

// GetGroup ...
func (c *fakeCache) GetGroup(ctx context.Context, s string) (models.Group, bool, error) {
	return c.fakeGroup, c.fakeFound, c.fakeError
}

// SetGroup ...
func (c *fakeCache) SetGroup(ctx context.Context, s string, g models.Group) error {
	return c.fakeError
}

func getEnvOrSkip(t *testing.T, envVar string) string {
	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s environment variable is empty, skipping.", envVar)
	}

	return v
}

func getAccessToken(ctx context.Context, tenantID, clientID, clientSecret, scope string) (*azcore.AccessToken, error) {
	tokenFilePath := fmt.Sprintf("../../tmp/test-token-file_%s", clientID)
	tokenFileExists := fileExists(tokenFilePath)
	token := &azcore.AccessToken{}

	generateNewToken := true
	if tokenFileExists {
		fileContent, err := getFileContent(tokenFilePath)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(fileContent, &token)
		if err != nil {
			return nil, err
		}

		if token.ExpiresOn.After(time.Now().Add(-5 * time.Minute)) {
			generateNewToken = false
		}
	}

	if generateNewToken {
		cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if err != nil {
			return nil, err
		}

		token, err := cred.GetToken(ctx, azcore.TokenRequestOptions{Scopes: []string{scope}})
		if err != nil {
			return nil, err
		}

		fileContents, err := json.Marshal(&token)
		if err != nil {
			return nil, err
		}

		err = ioutil.WriteFile(tokenFilePath, fileContents, 0644)
		if err != nil {
			return nil, err
		}

		return token, nil
	}

	return token, nil
}

func getFileContent(s string) ([]byte, error) {
	file, err := os.Open(s)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func fileExists(s string) bool {
	_, err := os.Stat(s)
	if err == nil {
		return true
	}

	return false
}
