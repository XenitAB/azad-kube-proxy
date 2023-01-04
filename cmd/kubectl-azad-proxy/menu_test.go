package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

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
