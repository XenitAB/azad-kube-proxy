package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/health"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
	"github.com/xenitab/azad-kube-proxy/pkg/user"
)

var (
	testFakeMaxGroups = 50
)

func TestNewHandlersClient(t *testing.T) {
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.Discard())
	testFakeCacheClient := newTestFakeCacheClient(t, "", "", nil, false, nil)
	testFakeUserClient := newTestFakeUserClient(t, "", "", nil, nil)
	testFakeHealthClient := newTestFakeHealthClient(t, true, nil, true, nil)

	cfg := &config.Config{
		AzureTenantID:     tenantID,
		KubernetesAPIHost: "fake-url",
		KubernetesAPITLS:  true,
	}

	_, err := NewHandlersClient(ctx, cfg, testFakeCacheClient, testFakeUserClient, testFakeHealthClient)
	require.NoError(t, err)
}

func TestReadinessHandler(t *testing.T) {
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.Discard())

	req, err := http.NewRequest("GET", "/readyz", nil)
	require.NoError(t, err)

	fakeURL, err := url.Parse("https://fake-url")
	require.NoError(t, err)

	cfg := config.Config{
		TenantID: tenantID,
		KubernetesConfig: config.KubernetesConfig{
			URL: fakeURL,
		},
	}

	testFakeCacheClient := newTestFakeCacheClient(t, "", "", nil, true, nil)
	testFakeUserClient := newTestFakeUserClient(t, "", "", nil, nil)

	cases := []struct {
		healthClient    health.ClientInterface
		expectedString  string
		expectedResCode int
	}{
		{
			healthClient:    newTestFakeHealthClient(t, true, nil, true, nil),
			expectedString:  `{"status": "ok"}`,
			expectedResCode: http.StatusOK,
		},
		{
			healthClient:    newTestFakeHealthClient(t, false, nil, false, nil),
			expectedString:  `{"status": "error"}`,
			expectedResCode: http.StatusInternalServerError,
		},
	}

	for _, c := range cases {
		proxyHandlers, err := NewHandlersClient(ctx, cfg, testFakeCacheClient, testFakeUserClient, c.healthClient)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/readyz", proxyHandlers.ReadinessHandler(ctx)).Methods("GET")
		router.ServeHTTP(rr, req)
		require.Equal(t, c.expectedResCode, rr.Code)
		require.Equal(t, c.expectedString, rr.Body.String())
	}
}

func TestLivenessHandler(t *testing.T) {
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.Discard())

	req, err := http.NewRequest("GET", "/healthz", nil)
	require.NoError(t, err)

	fakeURL, err := url.Parse("https://fake-url")
	require.NoError(t, err)

	cfg := config.Config{
		TenantID: tenantID,
		KubernetesConfig: config.KubernetesConfig{
			URL: fakeURL,
		},
	}

	testFakeCacheClient := newTestFakeCacheClient(t, "", "", nil, true, nil)
	testFakeUserClient := newTestFakeUserClient(t, "", "", nil, nil)

	cases := []struct {
		healthClient    health.ClientInterface
		expectedString  string
		expectedResCode int
	}{
		{
			healthClient:    newTestFakeHealthClient(t, true, nil, true, nil),
			expectedString:  `{"status": "ok"}`,
			expectedResCode: http.StatusOK,
		},
		{
			healthClient:    newTestFakeHealthClient(t, false, nil, false, nil),
			expectedString:  `{"status": "error"}`,
			expectedResCode: http.StatusInternalServerError,
		},
	}

	for _, c := range cases {
		proxyHandlers, err := NewHandlersClient(ctx, cfg, testFakeCacheClient, testFakeUserClient, c.healthClient)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/healthz", proxyHandlers.LivenessHandler(ctx)).Methods("GET")
		router.ServeHTTP(rr, req)
		require.Equal(t, c.expectedResCode, rr.Code)
		require.Equal(t, c.expectedString, rr.Body.String())
	}
}

func TestAzadKubeProxyHandler(t *testing.T) {
	clientID := testGetEnvOrSkip(t, "CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	spClientID := testGetEnvOrSkip(t, "TEST_USER_SP_CLIENT_ID")
	spClientSecret := testGetEnvOrSkip(t, "TEST_USER_SP_CLIENT_SECRET")
	spResource := testGetEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

	ctx := logr.NewContext(context.Background(), logr.Discard())

	token := testGetAccessToken(t, ctx, tenantID, spClientID, spClientSecret, fmt.Sprintf("%s/.default", spResource))

	memCacheClient, err := cache.NewMemoryCache(5 * time.Minute)
	require.NoError(t, err)
	testFakeCacheClient := newTestFakeCacheClient(t, "", "", nil, false, nil)
	testFakeUserClient := newTestFakeUserClient(t, "", "", nil, nil)
	testFakeHealthClient := newTestFakeHealthClient(t, true, nil, true, nil)

	fakeBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{\"fake\": true}"))
	}))
	defer fakeBackend.Close()
	fakeBackendURL, err := url.Parse(fakeBackend.URL)
	require.NoError(t, err)

	cfg := config.Config{
		ClientID:             clientID,
		ClientSecret:         clientSecret,
		TenantID:             tenantID,
		AzureADMaxGroupCount: testFakeMaxGroups,
		GroupIdentifier:      models.NameGroupIdentifier,
		KubernetesConfig: config.KubernetesConfig{
			URL:   fakeBackendURL,
			Token: "fake-token",
		},
	}

	cases := []struct {
		testDescription     string
		request             *http.Request
		config              config.Config
		configFunction      func(oldConfig config.Config) config.Config
		cacheClient         cache.ClientInterface
		cacheFunction       func(oldCacheClient cache.ClientInterface) cache.ClientInterface
		userClient          user.ClientInterface
		userFunction        func(oldUserClient user.ClientInterface) user.ClientInterface
		expectedResCode     int
		expectedResBody     string
		expectedErrContains string
	}{
		{
			testDescription: "working token, fake user client",
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {fmt.Sprintf("Bearer %s", token.Token)},
				},
			},
			config:          cfg,
			cacheClient:     memCacheClient,
			userClient:      testFakeUserClient,
			expectedResCode: http.StatusOK,
			expectedResBody: `{"fake": true}`,
		},
		{
			testDescription: "no token",
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {"Bearer"},
				},
			},
			config:          cfg,
			cacheClient:     memCacheClient,
			expectedResCode: http.StatusBadRequest,
			expectedResBody: "",
		},
		{
			testDescription: "fake token",
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {"Bearer fake-token"},
				},
			},
			config:          cfg,
			cacheClient:     memCacheClient,
			userClient:      testFakeUserClient,
			expectedResCode: http.StatusUnauthorized,
			expectedResBody: "",
		},
		{
			testDescription: "working token, fake user client and cache",
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {fmt.Sprintf("Bearer %s", token.Token)},
				},
			},
			config:          cfg,
			cacheClient:     testFakeCacheClient,
			userClient:      testFakeUserClient,
			expectedResCode: http.StatusOK,
			expectedResBody: `{"fake": true}`,
		},
		{
			testDescription: "fake token",
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {"Bearer fake-token"},
				},
			},
			config:          cfg,
			cacheClient:     testFakeCacheClient,
			userClient:      testFakeUserClient,
			expectedResCode: http.StatusUnauthorized,
			expectedResBody: "",
		},
		{
			testDescription: "working token, error from cache",
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
			cacheClient:         newTestFakeCacheClient(t, "", "", nil, true, errors.New("Fake error")),
			userClient:          testFakeUserClient,
			expectedResCode:     http.StatusInternalServerError,
			expectedErrContains: "Unexpected error",
		},
		{
			testDescription: "working token, with imperonate-user header first",
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
			cacheClient:         testFakeCacheClient,
			userClient:          testFakeUserClient,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "User unauthorized",
		},
		{
			testDescription: "working token, with imperonate-user header last",
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
			cacheClient:         testFakeCacheClient,
			userClient:          testFakeUserClient,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "User unauthorized",
		},
		{
			testDescription: "working token, with imperonate-user header and fake-header",
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
			cacheClient:         testFakeCacheClient,
			userClient:          testFakeUserClient,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "User unauthorized",
		},
		{
			testDescription: "working token, with imperonate-group header and fake-header",
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
			cacheClient:         testFakeCacheClient,
			userClient:          testFakeUserClient,
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "User unauthorized",
		},
		{
			testDescription: "working token, userClient error",
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
			cacheClient:         testFakeCacheClient,
			userClient:          newTestFakeUserClient(t, "", "", nil, errors.New("fake error")),
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "Unable to get user",
		},
		{
			testDescription: "working token, with multiple fake groups",
			request: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Path: "/",
				},
				Header: map[string][]string{
					"Authorization": {fmt.Sprintf("Bearer %s", token.Token)},
				},
			},
			config:      cfg,
			cacheClient: testFakeCacheClient,
			userClient:  testFakeUserClient,
			userFunction: func(oldUserClient user.ClientInterface) user.ClientInterface {
				i := 1
				groups := []models.Group{}
				for i < testFakeMaxGroups+1 {
					groups = append(groups, models.Group{
						Name: fmt.Sprintf("group-%d", i),
					})
					i++
				}

				return newTestFakeUserClient(t, "", "", groups, nil)
			},
			expectedResCode:     http.StatusForbidden,
			expectedErrContains: "Too many groups",
		},
		{
			testDescription: "working token, using ObjectIDGroupIdentifier",
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
			cacheClient:     memCacheClient,
			userClient:      testFakeUserClient,
			expectedResCode: http.StatusOK,
			expectedResBody: `{"fake": true}`,
		},
		{
			testDescription: "working token, with wrong GroupIdentifier",
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
			userClient:          testFakeUserClient,
			expectedResCode:     http.StatusInternalServerError,
			expectedErrContains: "Unexpected error",
		},
	}

	for i, c := range cases {
		t.Logf("Test #%d: %s", i, c.testDescription)
		if c.configFunction != nil {
			c.config = c.configFunction(c.config)
		}

		if c.cacheFunction != nil {
			c.cacheClient = c.cacheFunction(c.cacheClient)
		}

		if c.userFunction != nil {
			c.userClient = c.userFunction(c.userClient)
		}

		proxyHandlers, err := NewHandlersClient(ctx, c.config, c.cacheClient, c.userClient, testFakeHealthClient)
		require.NoError(t, err)

		proxy := httputil.NewSingleHostReverseProxy(c.config.KubernetesConfig.URL)
		proxy.ErrorHandler = proxyHandlers.ErrorHandler(ctx)
		rr := httptest.NewRecorder()
		router := mux.NewRouter()

		oidcHandler := NewOIDCHandler(proxyHandlers.AzadKubeProxyHandler(ctx, proxy), tenantID, clientID)
		router.PathPrefix("/").Handler(oidcHandler)

		router.ServeHTTP(rr, c.request)
		require.Equal(t, c.expectedResCode, rr.Code)

		if c.expectedErrContains == "" {
			require.Equal(t, c.expectedResBody, rr.Body.String())
		} else {
			require.Contains(t, rr.Body.String(), c.expectedErrContains)
		}
	}
}

type testFakeUserClient struct {
	fakeError error
	fakeUser  models.User
	fakeGroup models.Group
	t         *testing.T
}

func newTestFakeUserClient(t *testing.T, username string, objectID string, groups []models.Group, fakeError error) *testFakeUserClient {
	t.Helper()

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
	return &testFakeUserClient{
		fakeError: fakeError,
		fakeUser: models.User{
			Username: username,
			ObjectID: objectID,
			Groups:   groups,
		},
		fakeGroup: groups[0],
		t:         t,
	}
}

func (client *testFakeUserClient) GetUser(ctx context.Context, username, objectID string) (models.User, error) {
	client.t.Helper()

	return client.fakeUser, client.fakeError
}

type testFakeCacheClient struct {
	fakeError error
	fakeFound bool
	fakeUser  models.User
	fakeGroup models.Group
	t         *testing.T
}

func newTestFakeCacheClient(t *testing.T, username string, objectID string, groups []models.Group, fakeFound bool, fakeError error) *testFakeCacheClient {
	t.Helper()

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

	return &testFakeCacheClient{
		fakeError: fakeError,
		fakeFound: fakeFound,
		fakeUser: models.User{
			Username: username,
			ObjectID: objectID,
			Groups:   groups,
		},
		fakeGroup: groups[0],
		t:         t,
	}
}

func (c *testFakeCacheClient) GetUser(ctx context.Context, s string) (models.User, bool, error) {
	c.t.Helper()

	return c.fakeUser, c.fakeFound, c.fakeError
}

func (c *testFakeCacheClient) SetUser(ctx context.Context, s string, u models.User) error {
	c.t.Helper()

	return c.fakeError
}

func (c *testFakeCacheClient) GetGroup(ctx context.Context, s string) (models.Group, bool, error) {
	c.t.Helper()

	return c.fakeGroup, c.fakeFound, c.fakeError
}

func (c *testFakeCacheClient) SetGroup(ctx context.Context, s string, g models.Group) error {
	c.t.Helper()

	return c.fakeError
}

type testFakeHealthClient struct {
	ready      bool
	readyError error
	live       bool
	liveError  error
	t          *testing.T
}

func newTestFakeHealthClient(t *testing.T, ready bool, readyError error, live bool, liveError error) health.ClientInterface {
	t.Helper()

	return &testFakeHealthClient{
		ready,
		readyError,
		live,
		liveError,
		t,
	}
}

func (client *testFakeHealthClient) Ready(ctx context.Context) (bool, error) {
	client.t.Helper()

	return client.ready, client.readyError
}

func (client *testFakeHealthClient) Live(ctx context.Context) (bool, error) {
	client.t.Helper()

	return client.live, client.liveError
}

func testGetEnvOrSkip(t *testing.T, envVar string) string {
	t.Helper()

	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s environment variable is empty, skipping.", envVar)
	}

	return v
}

func testGetAccessToken(t *testing.T, ctx context.Context, tenantID, clientID, clientSecret, scope string) *azcore.AccessToken {
	t.Helper()

	tokenFilePath := fmt.Sprintf("../../tmp/test-token-file_%s", clientID)
	tokenFileExists := testFileExists(t, tokenFilePath)
	token := &azcore.AccessToken{}

	generateNewToken := true
	if tokenFileExists {
		fileContent := testGetFileContent(t, tokenFilePath)
		err := json.Unmarshal(fileContent, &token)
		require.NoError(t, err)

		if token.ExpiresOn.After(time.Now().Add(-5 * time.Minute)) {
			generateNewToken = false
		}
	}

	if generateNewToken {
		cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		require.NoError(t, err)

		token, err := cred.GetToken(ctx, azpolicy.TokenRequestOptions{Scopes: []string{scope}})
		require.NoError(t, err)

		fileContents, err := json.Marshal(&token)
		require.NoError(t, err)

		err = os.WriteFile(tokenFilePath, fileContents, 0600)
		require.NoError(t, err)

		return &token
	}

	return token
}

func testGetFileContent(t *testing.T, s string) []byte {
	t.Helper()

	file, err := os.Open(s)
	require.NoError(t, err)

	defer file.Close()

	bytes, err := io.ReadAll(file)
	require.NoError(t, err)

	return bytes
}

func testFileExists(t *testing.T, s string) bool {
	t.Helper()

	f, err := os.Stat(s)
	if err != nil {
		return false
	}

	if f.IsDir() {
		return false
	}

	return true
}
