package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func TestRunMenu(t *testing.T) {
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	clientID := testGetEnvOrSkip(t, "CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "CLIENT_SECRET")
	resource := testGetEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

	ctx := logr.NewContext(context.Background(), logr.Discard())

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tokenCacheDir := tmpDir
	kubeConfigFile := fmt.Sprintf("%s/kubeconfig", tmpDir)

	cfg := menuConfig{
		Output:                "JSON",
		AzureTenantID:         tenantID,
		AzureClientID:         clientID,
		AzureClientSecret:     clientSecret,
		ClusterName:           "ze-cluster",
		ProxyURL:              srv.URL,
		Resource:              resource,
		KubeConfig:            kubeConfigFile,
		TokenCacheDir:         tokenCacheDir,
		Overwrite:             false,
		TLSInsecureSkipVerify: true,
	}

	authCfg := authConfig{
		excludeAzureCLIAuth:    true,
		excludeEnvironmentAuth: false,
		excludeMSIAuth:         true,
	}

	err = runMenu(ctx, cfg, authCfg, newtestFakePromptClient(t, true, nil, nil))
	require.Error(t, err)
}

func TestMenu(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	cases := []struct {
		menuClient  *MenuClient
		expectedErr bool
	}{
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, nil),
				generateClient: newTestFakeGenerateClient(t, nil),
				promptClient:   newtestFakePromptClient(t, true, nil, nil),
			},
			expectedErr: false,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, nil),
				generateClient: newTestFakeGenerateClient(t, newCustomError(errorTypeOverwriteConfig, fmt.Errorf("Fake error"))),
				promptClient:   newtestFakePromptClient(t, true, nil, nil),
			},
			expectedErr: false,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, nil),
				generateClient: newTestFakeGenerateClient(t, newCustomError(errorTypeUnknown, fmt.Errorf("Fake error"))),
				promptClient:   newtestFakePromptClient(t, true, nil, nil),
			},
			expectedErr: true,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, fmt.Errorf("Fake error")),
				generateClient: newTestFakeGenerateClient(t, nil),
				promptClient:   newtestFakePromptClient(t, true, nil, nil),
			},
			expectedErr: true,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, nil),
				generateClient: newTestFakeGenerateClient(t, nil),
				promptClient:   newtestFakePromptClient(t, true, fmt.Errorf("Fake error"), nil),
			},
			expectedErr: true,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, nil),
				generateClient: newTestFakeGenerateClient(t, nil),
				promptClient:   newtestFakePromptClient(t, true, nil, fmt.Errorf("Fake error")),
			},
			expectedErr: false,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, nil),
				generateClient: newTestFakeGenerateClient(t, newCustomError(errorTypeOverwriteConfig, fmt.Errorf("Fake error"))),
				promptClient:   newtestFakePromptClient(t, true, nil, fmt.Errorf("Fake error")),
			},
			expectedErr: true,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, nil),
				generateClient: newTestFakeGenerateClient(t, newCustomError(errorTypeOverwriteConfig, fmt.Errorf("Fake error"))),
				promptClient:   newtestFakePromptClient(t, false, nil, nil),
			},
			expectedErr: false,
		},
	}

	for _, c := range cases {
		err := c.menuClient.Menu(ctx)
		if c.expectedErr {
			require.Error(t, err)
			continue
		}

		require.NoError(t, err)
	}
}

type testFakeDiscoverClient struct {
	fakeError error
	t         *testing.T
}

func newTestFakeDiscoverClient(t *testing.T, fakeError error) DiscoverInterface {
	t.Helper()

	return &testFakeDiscoverClient{
		fakeError: fakeError,
		t:         t,
	}
}

func (client *testFakeDiscoverClient) Discover(ctx context.Context) (string, error) {
	client.t.Helper()

	return "", nil
}
func (client *testFakeDiscoverClient) Run(ctx context.Context) ([]discover, error) {
	client.t.Helper()

	fakseClusters := []discover{
		{
			ClusterName: "dev",
			Resource:    "https://dev.example.com",
			ProxyURL:    "https://dev.example.com",
		},
		{
			ClusterName: "qa",
			Resource:    "https://qa.example.com",
			ProxyURL:    "https://qa.example.com",
		},
		{
			ClusterName: "prod",
			Resource:    "https://prod.example.com",
			ProxyURL:    "https://prod.example.com",
		},
	}

	return fakseClusters, client.fakeError
}

type testFakeGenerateClient struct {
	fakeError error
	overwrite bool
	t         *testing.T
}

func newTestFakeGenerateClient(t *testing.T, fakeError error) GenerateInterface {
	t.Helper()

	return &testFakeGenerateClient{
		fakeError: fakeError,
		overwrite: false,
		t:         t,
	}
}

func (client *testFakeGenerateClient) Generate(ctx context.Context) error {
	client.t.Helper()

	if toCustomError(client.fakeError).errorType == errorTypeOverwriteConfig {
		if !client.overwrite {
			return client.fakeError
		}
		return nil
	}

	return client.fakeError
}

func (client *testFakeGenerateClient) Merge(new GenerateClient) {
	client.t.Helper()

	if new.overwrite != client.overwrite {
		client.overwrite = new.overwrite
	}
}

type testFakePromptClient struct {
	userSelectedOverwrite bool
	selectClusterError    error
	overwriteConfigError  error
	t                     *testing.T
}

func newtestFakePromptClient(t *testing.T, userSelectedOverwrite bool, selectClusterError error, overwriteConfigError error) promptInterface {
	t.Helper()

	return &testFakePromptClient{
		userSelectedOverwrite: userSelectedOverwrite,
		selectClusterError:    selectClusterError,
		overwriteConfigError:  overwriteConfigError,
		t:                     t,
	}
}

func (client *testFakePromptClient) selectCluster(apps []discover) (discover, error) {
	client.t.Helper()

	if len(apps) == 0 {
		return discover{}, fmt.Errorf("Empty array")
	}
	return apps[0], client.selectClusterError
}

func (client *testFakePromptClient) overwriteConfig() (bool, error) {
	client.t.Helper()

	return client.userSelectedOverwrite, client.overwriteConfigError
}
