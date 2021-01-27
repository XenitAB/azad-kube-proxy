package handlers

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/coreos/go-oidc"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/claims"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/user"
)

var (
	fakeMaxGroups = 50
)

func TestNewHandlersClient(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	fakeClaimsClient := newFakeClaimsClient(nil, nil, claims.AzureClaims{}, &oidc.IDTokenVerifier{})
	fakeCacheClient := newFakeCacheClient("", "", nil, false, nil)
	fakeUserClient := newFakeUserClient("", "", nil, nil)

	_, err := NewHandlersClient(ctx, config.Config{TenantID: tenantID}, fakeCacheClient, fakeUserClient, fakeClaimsClient)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	fakeClaimsClient = newFakeClaimsClient(nil, errors.New("fake error"), claims.AzureClaims{}, &oidc.IDTokenVerifier{})
	_, err = NewHandlersClient(ctx, config.Config{TenantID: tenantID}, fakeCacheClient, fakeUserClient, fakeClaimsClient)
	if !strings.Contains(err.Error(), "fake error") {
		t.Errorf("Expected err to contain 'fake error' but it was %q", err)
	}
}

func TestReadinessHandler(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	req, err := http.NewRequest("GET", "/readyz", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	config := config.Config{
		TenantID: tenantID,
	}

	fakeCacheClient := newFakeCacheClient("", "", nil, true, nil)
	fakeUserClient := newFakeUserClient("", "", nil, nil)
	claimsClient := claims.NewClaimsClient()
	proxyHandlers, err := NewHandlersClient(ctx, config, fakeCacheClient, fakeUserClient, claimsClient)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

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
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	config := config.Config{
		TenantID: tenantID,
	}

	fakeCacheClient := newFakeCacheClient("", "", nil, true, nil)
	fakeUserClient := newFakeUserClient("", "", nil, nil)
	claimsClient := claims.NewClaimsClient()
	proxyHandlers, err := NewHandlersClient(ctx, config, fakeCacheClient, fakeUserClient, claimsClient)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

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

	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	token, err := getAccessToken(ctx, tenantID, spClientID, spClientSecret, fmt.Sprintf("%s/.default", spResource))
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	memCacheClient, err := cache.NewMemoryCache(5*time.Minute, 10*time.Minute)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	claimsClient := claims.NewClaimsClient()
	fakeCacheClient := newFakeCacheClient("", "", nil, false, nil)
	fakeUserClient := newFakeUserClient("", "", nil, nil)

	fakeBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{\"fake\": true}"))
	}))
	defer fakeBackend.Close()
	fakeBackendURL, err := url.Parse(fakeBackend.URL)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	cfg := config.Config{
		ClientID:             clientID,
		ClientSecret:         clientSecret,
		TenantID:             tenantID,
		CacheEngine:          models.MemoryCacheEngine,
		AzureADMaxGroupCount: fakeMaxGroups,
		GroupIdentifier:      models.NameGroupIdentifier,
		KubernetesConfig: config.KubernetesConfig{
			URL:   fakeBackendURL,
			Token: "fake-token",
		},
	}

	cases := []struct {
		request             *http.Request
		config              config.Config
		configFunction      func(oldConfig config.Config) config.Config
		cacheClient         cache.ClientInterface
		cacheFunction       func(oldCacheClient cache.ClientInterface) cache.ClientInterface
		claimsClient        claims.ClientInterface
		claimsFunction      func(oldClaimsClient claims.ClientInterface) claims.ClientInterface
		userClient          user.ClientInterface
		userFunction        func(oldUserClient user.ClientInterface) user.ClientInterface
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
			config:              cfg,
			cacheClient:         memCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
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
			config:              cfg,
			cacheClient:         memCacheClient,
			claimsClient:        claimsClient,
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
			config:              cfg,
			cacheClient:         memCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
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
					"Authorization": {fmt.Sprintf("Bearer %s", token.Token)},
				},
			},
			config:              cfg,
			cacheClient:         fakeCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
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
					"Authorization": {"Bearer fake-token"},
				},
			},
			config:              cfg,
			cacheClient:         fakeCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
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
					"Authorization": {fmt.Sprintf("Bearer %s", token.Token)},
				},
			},
			config:              cfg,
			cacheClient:         newFakeCacheClient("", "", nil, true, errors.New("Fake error")),
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
			expectedResCode:     http.StatusInternalServerError,
			expectedErrContains: "Unexpected error",
		},
		{
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Impersonate-User": {"this-should-not-work"},
					"Authorization":    {fmt.Sprintf("Bearer %s", token.Token)},
				},
			},
			config:              cfg,
			cacheClient:         fakeCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "User unauthorized",
		},
		{
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization":    {fmt.Sprintf("Bearer %s", token.Token)},
					"Impersonate-User": {"this-should-not-work"},
				},
			},
			config:              cfg,
			cacheClient:         fakeCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "User unauthorized",
		},
		{
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Fake-Header":      {"fake"},
					"Authorization":    {fmt.Sprintf("Bearer %s", token.Token)},
					"Impersonate-User": {"this-should-not-work"},
				},
			},
			config:              cfg,
			cacheClient:         fakeCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "User unauthorized",
		},
		{
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Fake-Header":       {"fake"},
					"Authorization":     {fmt.Sprintf("Bearer %s", token.Token)},
					"Impersonate-Group": {"this-should-not-work"},
				},
			},
			config:              cfg,
			cacheClient:         fakeCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "User unauthorized",
		},
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
			config:              cfg,
			cacheClient:         fakeCacheClient,
			claimsClient:        newFakeClaimsClient(errors.New("fake error"), nil, claims.AzureClaims{}, nil),
			userClient:          fakeUserClient,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "Unable to get claims",
		},
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
			config:              cfg,
			cacheClient:         fakeCacheClient,
			claimsClient:        claimsClient,
			userClient:          newFakeUserClient("", "", nil, errors.New("fake error")),
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "Unable to get user",
		},
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
			config:       cfg,
			cacheClient:  fakeCacheClient,
			claimsClient: claimsClient,
			userClient:   fakeUserClient,
			userFunction: func(oldUserClient user.ClientInterface) user.ClientInterface {
				i := 1
				groups := []models.Group{}
				for i < fakeMaxGroups+1 {
					groups = append(groups, models.Group{
						Name: fmt.Sprintf("group-%d", i),
					})
					i++
				}

				return newFakeUserClient("", "", groups, nil)
			},
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "Too many groups",
		},
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
			config: cfg,
			configFunction: func(oldConfig config.Config) config.Config {
				oldConfig.GroupIdentifier = models.ObjectIDGroupIdentifier
				return oldConfig
			},
			cacheClient:         memCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
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
					"Authorization": {fmt.Sprintf("Bearer %s", token.Token)},
				},
			},
			config: cfg,
			configFunction: func(oldConfig config.Config) config.Config {
				oldConfig.GroupIdentifier = "DUMMY"
				return oldConfig
			},
			cacheClient:         memCacheClient,
			claimsClient:        claimsClient,
			userClient:          fakeUserClient,
			expectedResCode:     http.StatusInternalServerError,
			expectedErrContains: "Unexpected error",
		},
	}

	for _, c := range cases {
		if c.configFunction != nil {
			c.config = c.configFunction(c.config)
		}

		if c.cacheFunction != nil {
			c.cacheClient = c.cacheFunction(c.cacheClient)
		}

		if c.claimsFunction != nil {
			c.claimsClient = c.claimsFunction(c.claimsClient)
		}

		if c.userFunction != nil {
			c.userClient = c.userFunction(c.userClient)
		}

		proxyHandlers, err := NewHandlersClient(ctx, c.config, c.cacheClient, c.userClient, c.claimsClient)
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		proxy := httputil.NewSingleHostReverseProxy(c.config.KubernetesConfig.URL)
		proxy.ErrorHandler = proxyHandlers.ErrorHandler(ctx)
		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.PathPrefix("/").HandlerFunc(proxyHandlers.AzadKubeProxyHandler(ctx, proxy))
		router.ServeHTTP(rr, c.request)

		if rr.Code != c.expectedResCode {
			t.Errorf("Handler returned unexpected status code.\nExpected: %d\nActual:   %d", c.expectedResCode, rr.Code)
		}

		expected := `{"fake": true}`
		if rr.Body.String() != expected && c.expectedErrContains == "" {
			t.Errorf("Handler returned unexpected body.\nExpected: %s\nActual:   %s", expected, rr.Body.String())
		}

		if c.expectedErrContains != "" {
			if !strings.Contains(rr.Body.String(), c.expectedErrContains) {
				t.Errorf("Handler returned unexpected body.\nExpected: %s\nActual:   %s", c.expectedErrContains, rr.Body.String())
			}
		}
	}
}

type fakeUserClient struct {
	fakeError error
	fakeUser  models.User
	fakeGroup models.Group
}

func newFakeUserClient(username string, objectID string, groups []models.Group, fakeError error) *fakeUserClient {
	if username == "" {
		username = "username"
	}
	if objectID == "" {
		objectID = "00000000-0000-0000-0000-000000000000"
	}
	if len(groups) == 0 {
		groups = []models.Group{
			{Name: "group"},
		}
	}
	return &fakeUserClient{
		fakeError: fakeError,
		fakeUser: models.User{
			Username: username,
			ObjectID: objectID,
			Groups:   groups,
		},
		fakeGroup: groups[0],
	}
}

func (client *fakeUserClient) GetUser(ctx context.Context, username, objectID string) (models.User, error) {
	return client.fakeUser, client.fakeError
}

type fakeCacheClient struct {
	fakeError error
	fakeFound bool
	fakeUser  models.User
	fakeGroup models.Group
}

func newFakeCacheClient(username string, objectID string, groups []models.Group, fakeFound bool, fakeError error) *fakeCacheClient {
	if username == "" {
		username = "username"
	}
	if objectID == "" {
		objectID = "00000000-0000-0000-0000-000000000000"
	}
	if len(groups) == 0 {
		groups = []models.Group{
			{Name: "group"},
		}
	}

	return &fakeCacheClient{
		fakeError: fakeError,
		fakeFound: fakeFound,
		fakeUser: models.User{
			Username: username,
			ObjectID: objectID,
			Groups:   groups,
		},
		fakeGroup: groups[0],
	}
}

func (c *fakeCacheClient) GetUser(ctx context.Context, s string) (models.User, bool, error) {
	return c.fakeUser, c.fakeFound, c.fakeError
}

func (c *fakeCacheClient) SetUser(ctx context.Context, s string, u models.User) error {
	return c.fakeError
}

func (c *fakeCacheClient) GetGroup(ctx context.Context, s string) (models.Group, bool, error) {
	return c.fakeGroup, c.fakeFound, c.fakeError
}

func (c *fakeCacheClient) SetGroup(ctx context.Context, s string, g models.Group) error {
	return c.fakeError
}

type fakeClaimsClient struct {
	fakeAzureClaims          claims.AzureClaims
	fakeOIDCVerifier         *oidc.IDTokenVerifier
	newClaimsFakeError       error
	getOIDCVerifierFakeError error
}

func newFakeClaimsClient(newClaimsFakeError error, getOIDCVerifierFakeError error, fakeAzureClaims claims.AzureClaims, fakeOIDCVerifier *oidc.IDTokenVerifier) *fakeClaimsClient {
	return &fakeClaimsClient{
		fakeAzureClaims:          fakeAzureClaims,
		fakeOIDCVerifier:         fakeOIDCVerifier,
		newClaimsFakeError:       newClaimsFakeError,
		getOIDCVerifierFakeError: getOIDCVerifierFakeError,
	}
}

func (client *fakeClaimsClient) NewClaims(t *oidc.IDToken) (claims.AzureClaims, error) {
	if client.newClaimsFakeError != nil {
		return claims.AzureClaims{}, client.newClaimsFakeError
	}
	if client.fakeAzureClaims.Issuer == "" {
		realClaimsClient := claims.NewClaimsClient()
		realClaims, err := realClaimsClient.NewClaims(t)
		if err != nil {
			return claims.AzureClaims{}, err
		}
		return realClaims, nil
	}

	return client.fakeAzureClaims, client.newClaimsFakeError
}

func (client *fakeClaimsClient) GetOIDCVerifier(ctx context.Context, tenantID, clientID string) (*oidc.IDTokenVerifier, error) {
	if client.getOIDCVerifierFakeError != nil {
		return nil, client.getOIDCVerifierFakeError
	}

	log := logr.FromContext(ctx)
	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		log.Error(err, "Unable to initiate OIDC provider")
		return nil, err
	}

	oidcConfig := &oidc.Config{
		ClientID: clientID,
	}

	verifier := provider.Verifier(oidcConfig)

	return verifier, nil
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
	return err == nil
}
