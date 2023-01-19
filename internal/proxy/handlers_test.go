package proxy

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
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	azpolicy "github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
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

	kubernetesAPITokenPath, cleanupFn := testGetKubernetesAPITokenPath(t)
	defer cleanupFn()

	cfg := &config{
		AzureTenantID:          tenantID,
		KubernetesAPIHost:      "fake-url",
		KubernetesAPITLS:       true,
		KubernetesAPITokenPath: kubernetesAPITokenPath,
		GroupIdentifier:        "NAME",
	}

	_, err := newHandlers(ctx, cfg, testFakeCacheClient, testFakeUserClient, testFakeHealthClient)
	require.NoError(t, err)
}

func TestReadinessHandler(t *testing.T) {
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.Discard())

	req, err := http.NewRequest("GET", "/readyz", nil)
	require.NoError(t, err)

	kubernetesAPITokenPath, cleanupFn := testGetKubernetesAPITokenPath(t)
	defer cleanupFn()

	cfg := &config{
		AzureTenantID:          tenantID,
		KubernetesAPIHost:      "fake-url",
		KubernetesAPITLS:       true,
		KubernetesAPITokenPath: kubernetesAPITokenPath,
		GroupIdentifier:        "NAME",
	}

	testFakeCacheClient := newTestFakeCacheClient(t, "", "", nil, true, nil)
	testFakeUserClient := newTestFakeUserClient(t, "", "", nil, nil)

	cases := []struct {
		healthClient    Health
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
		proxyHandlers, err := newHandlers(ctx, cfg, testFakeCacheClient, testFakeUserClient, c.healthClient)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/readyz", proxyHandlers.readiness(ctx)).Methods("GET")
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

	kubernetesAPITokenPath, cleanupFn := testGetKubernetesAPITokenPath(t)
	defer cleanupFn()

	cfg := &config{
		AzureTenantID:          tenantID,
		KubernetesAPIHost:      "fake-url",
		KubernetesAPITLS:       true,
		KubernetesAPITokenPath: kubernetesAPITokenPath,
		GroupIdentifier:        "NAME",
	}

	testFakeCacheClient := newTestFakeCacheClient(t, "", "", nil, true, nil)
	testFakeUserClient := newTestFakeUserClient(t, "", "", nil, nil)

	cases := []struct {
		healthClient    Health
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
		proxyHandlers, err := newHandlers(ctx, cfg, testFakeCacheClient, testFakeUserClient, c.healthClient)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/healthz", proxyHandlers.liveness(ctx)).Methods("GET")
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

	memCacheClient, err := newMemoryCache(5 * time.Minute)
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
	fakeBackendPort, err := strconv.Atoi(fakeBackendURL.Port())
	require.NoError(t, err)

	kubernetesAPITokenPath, cleanupFn := testGetKubernetesAPITokenPath(t)
	defer cleanupFn()

	cfg := &config{
		AzureClientID:          clientID,
		AzureClientSecret:      clientSecret,
		AzureTenantID:          tenantID,
		AzureADMaxGroupCount:   testFakeMaxGroups,
		GroupIdentifier:        "NAME",
		KubernetesAPIHost:      fakeBackendURL.Hostname(),
		KubernetesAPIPort:      fakeBackendPort,
		KubernetesAPITLS:       false,
		KubernetesAPITokenPath: kubernetesAPITokenPath,
	}

	cases := []struct {
		testDescription     string
		request             *http.Request
		config              *config
		configFunction      func(oldConfig config) config
		cacheClient         Cache
		cacheFunction       func(oldCacheClient Cache) Cache
		userClient          User
		userFunction        func(oldUserClient User) User
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
			userFunction: func(oldUserClient User) User {
				i := 1
				groups := []groupModel{}
				for i < testFakeMaxGroups+1 {
					groups = append(groups, groupModel{
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
			configFunction: func(oldConfig config) config {
				oldConfig.GroupIdentifier = "OBJECTID"
				return oldConfig
			},
			cacheClient:     memCacheClient,
			userClient:      testFakeUserClient,
			expectedResCode: http.StatusOK,
			expectedResBody: `{"fake": true}`,
		},
	}

	for i, c := range cases {
		t.Logf("Test #%d: %s", i, c.testDescription)
		if c.configFunction != nil {
			tmpCfg := c.configFunction(*c.config)
			c.config = &tmpCfg
		}

		if c.cacheFunction != nil {
			c.cacheClient = c.cacheFunction(c.cacheClient)
		}

		if c.userFunction != nil {
			c.userClient = c.userFunction(c.userClient)
		}

		proxyHandlers, err := newHandlers(ctx, c.config, c.cacheClient, c.userClient, testFakeHealthClient)
		require.NoError(t, err)

		kubernetesAPIUrl := testGetKubernetesAPIUrl(t, c.config.KubernetesAPIHost, c.config.KubernetesAPIPort, c.config.KubernetesAPITLS)
		proxy := httputil.NewSingleHostReverseProxy(kubernetesAPIUrl)
		proxy.ErrorHandler = proxyHandlers.error(ctx)
		rr := httptest.NewRecorder()
		router := mux.NewRouter()

		oidcHandler := newOIDCHandler(proxyHandlers.proxy(ctx, proxy), tenantID, clientID)
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

func testGetKubernetesAPITokenPath(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	cleanupFn := func() { os.RemoveAll(tmpDir) }
	kubernetesAPITokenPath := filepath.Clean(fmt.Sprintf("%s/kubernetes-token", tmpDir))
	err = os.WriteFile(kubernetesAPITokenPath, []byte("fake-token"), 0600)
	require.NoError(t, err)
	return kubernetesAPITokenPath, cleanupFn
}

type testFakeUserClient struct {
	fakeError error
	fakeUser  userModel
	fakeGroup groupModel
	t         *testing.T
}

func newTestFakeUserClient(t *testing.T, username string, objectID string, groups []groupModel, fakeError error) *testFakeUserClient {
	t.Helper()

	if username == "" {
		username = "username"
	}
	if objectID == "" {
		objectID = "00000000-0000-0000-0000-000000000000"
	}
	if len(groups) == 0 {
		groups = []groupModel{
			{Name: "group"},
		}
	}
	return &testFakeUserClient{
		fakeError: fakeError,
		fakeUser: userModel{
			Username: username,
			ObjectID: objectID,
			Groups:   groups,
		},
		fakeGroup: groups[0],
		t:         t,
	}
}

func (client *testFakeUserClient) getUser(ctx context.Context, username, objectID string) (userModel, error) {
	client.t.Helper()

	return client.fakeUser, client.fakeError
}

type testFakeCacheClient struct {
	fakeError error
	fakeFound bool
	fakeUser  userModel
	fakeGroup groupModel
	t         *testing.T
}

func newTestFakeCacheClient(t *testing.T, username string, objectID string, groups []groupModel, fakeFound bool, fakeError error) *testFakeCacheClient {
	t.Helper()

	if username == "" {
		username = "username"
	}
	if objectID == "" {
		objectID = "00000000-0000-0000-0000-000000000000"
	}
	if len(groups) == 0 {
		groups = []groupModel{
			{Name: "group"},
		}
	}

	return &testFakeCacheClient{
		fakeError: fakeError,
		fakeFound: fakeFound,
		fakeUser: userModel{
			Username: username,
			ObjectID: objectID,
			Groups:   groups,
		},
		fakeGroup: groups[0],
		t:         t,
	}
}

func (c *testFakeCacheClient) GetUser(ctx context.Context, s string) (userModel, bool, error) {
	c.t.Helper()

	return c.fakeUser, c.fakeFound, c.fakeError
}

func (c *testFakeCacheClient) SetUser(ctx context.Context, s string, u userModel) error {
	c.t.Helper()

	return c.fakeError
}

func (c *testFakeCacheClient) GetGroup(ctx context.Context, s string) (groupModel, bool, error) {
	c.t.Helper()

	return c.fakeGroup, c.fakeFound, c.fakeError
}

func (c *testFakeCacheClient) SetGroup(ctx context.Context, s string, g groupModel) error {
	c.t.Helper()

	return c.fakeError
}

type testFakeHealthClient struct {
	isReady    bool
	readyError error
	isLive     bool
	liveError  error
	t          *testing.T
}

func newTestFakeHealthClient(t *testing.T, isReady bool, readyError error, isLive bool, liveError error) *testFakeHealthClient {
	t.Helper()

	return &testFakeHealthClient{
		isReady,
		readyError,
		isLive,
		liveError,
		t,
	}
}

func (client *testFakeHealthClient) ready(ctx context.Context) (bool, error) {
	client.t.Helper()

	return client.isReady, client.readyError
}

func (client *testFakeHealthClient) live(ctx context.Context) (bool, error) {
	client.t.Helper()

	return client.isLive, client.liveError
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

func testGetKubernetesAPIUrl(t *testing.T, host string, port int, tls bool) *url.URL {
	t.Helper()

	httpScheme := testGetHTTPScheme(t, tls)
	u, err := url.Parse(fmt.Sprintf("%s://%s:%d", httpScheme, host, port))
	require.NoError(t, err)

	return u
}

func testGetHTTPScheme(t *testing.T, tls bool) string {
	t.Helper()

	if tls {
		return "https"
	}

	return "http"
}
