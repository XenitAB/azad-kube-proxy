package actions

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/customerrors"
)

func TestNewMenuClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	restoreAzureCLIAuth := testTempChangeEnv(t, "EXCLUDE_AZURE_CLI_AUTH", "true")
	defer restoreAzureCLIAuth()

	menuFlags, err := MenuFlags(ctx)
	require.NoError(t, err)

	app := &cli.App{
		Name:  "test",
		Usage: "test",
		Commands: []*cli.Command{
			{
				Name:    "test",
				Aliases: []string{"t"},
				Usage:   "test",
				Flags:   menuFlags,
				Action: func(c *cli.Context) error {
					_, err := NewMenuClient(ctx, c)
					if err != nil {
						return err
					}
					return nil
				},
			},
		},
	}

	app.Writer = &bytes.Buffer{}
	app.ErrWriter = &bytes.Buffer{}
	err = app.Run([]string{"fake-binary", "test"})
	require.NoError(t, err)
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
				generateClient: newTestFakeGenerateClient(t, customerrors.New(customerrors.ErrorTypeOverwriteConfig, fmt.Errorf("Fake error"))),
				promptClient:   newtestFakePromptClient(t, true, nil, nil),
			},
			expectedErr: false,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, nil),
				generateClient: newTestFakeGenerateClient(t, customerrors.New(customerrors.ErrorTypeUnknown, fmt.Errorf("Fake error"))),
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
				generateClient: newTestFakeGenerateClient(t, customerrors.New(customerrors.ErrorTypeOverwriteConfig, fmt.Errorf("Fake error"))),
				promptClient:   newtestFakePromptClient(t, true, nil, fmt.Errorf("Fake error")),
			},
			expectedErr: true,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newTestFakeDiscoverClient(t, nil),
				generateClient: newTestFakeGenerateClient(t, customerrors.New(customerrors.ErrorTypeOverwriteConfig, fmt.Errorf("Fake error"))),
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

func TestMergeFlags(t *testing.T) {
	cases := []struct {
		a              []cli.Flag
		b              []cli.Flag
		expectedLength int
	}{
		{
			a:              []cli.Flag{},
			b:              []cli.Flag{},
			expectedLength: 0,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			expectedLength: 2,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
			},
			expectedLength: 1,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			expectedLength: 2,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
				&cli.StringFlag{
					Name: "flag3",
				},
			},
			expectedLength: 3,
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
				&cli.StringFlag{
					Name: "flag3",
				},
			},
			b: []cli.Flag{
				&cli.StringFlag{
					Name: "flag1",
				},
				&cli.StringFlag{
					Name: "flag2",
				},
			},
			expectedLength: 3,
		},
	}

	for _, c := range cases {
		flags := mergeFlags(c.a, c.b)
		require.Len(t, flags, c.expectedLength)
	}
}

func TestUnrequireFlags(t *testing.T) {
	cases := []struct {
		a []cli.Flag
	}{
		{
			a: []cli.Flag{},
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name:     "flag1",
					Required: true,
				},
			},
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name:     "flag1",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "flag2",
					Required: true,
				},
			},
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name:     "flag1",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "flag2",
					Required: false,
				},
			},
		},
		{
			a: []cli.Flag{
				&cli.StringFlag{
					Name:     "flag1",
					Required: false,
				},
				&cli.StringFlag{
					Name:     "flag2",
					Required: false,
				},
				&cli.StringFlag{
					Name:     "flag3",
					Required: true,
				},
			},
		},
	}

	for _, c := range cases {
		flags := unrequireFlags(c.a)
		for _, flag := range flags {
			require.False(t, flag.(*cli.StringFlag).Required)
		}
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

type testGenerateInterface interface {
	Generate(ctx context.Context) error
	Merge(new GenerateClient)
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

	if customerrors.To(client.fakeError).ErrorType == customerrors.ErrorTypeOverwriteConfig {
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
