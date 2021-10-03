package actions

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/customerrors"
)

func TestNewMenuClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	restoreAzureCLIAuth := tempChangeEnv("EXCLUDE_AZURE_CLI_AUTH", "true")
	defer restoreAzureCLIAuth()

	app := &cli.App{
		Name:  "test",
		Usage: "test",
		Commands: []*cli.Command{
			{
				Name:    "test",
				Aliases: []string{"t"},
				Usage:   "test",
				Flags:   MenuFlags(ctx),
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
	err := app.Run([]string{"fake-binary", "test"})
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
}

func TestMenu(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	cases := []struct {
		menuClient  *MenuClient
		expectedErr bool
	}{
		{
			menuClient: &MenuClient{
				discoverClient: newFakeDiscoverClient(nil),
				generateClient: newFakeGenerateClient(nil),
				promptClient:   newFakePromptClient(true, nil, nil),
			},
			expectedErr: false,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newFakeDiscoverClient(nil),
				generateClient: newFakeGenerateClient(customerrors.New(customerrors.ErrorTypeOverwriteConfig, fmt.Errorf("Fake error"))),
				promptClient:   newFakePromptClient(true, nil, nil),
			},
			expectedErr: false,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newFakeDiscoverClient(nil),
				generateClient: newFakeGenerateClient(customerrors.New(customerrors.ErrorTypeUnknown, fmt.Errorf("Fake error"))),
				promptClient:   newFakePromptClient(true, nil, nil),
			},
			expectedErr: true,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newFakeDiscoverClient(fmt.Errorf("Fake error")),
				generateClient: newFakeGenerateClient(nil),
				promptClient:   newFakePromptClient(true, nil, nil),
			},
			expectedErr: true,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newFakeDiscoverClient(nil),
				generateClient: newFakeGenerateClient(nil),
				promptClient:   newFakePromptClient(true, fmt.Errorf("Fake error"), nil),
			},
			expectedErr: true,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newFakeDiscoverClient(nil),
				generateClient: newFakeGenerateClient(nil),
				promptClient:   newFakePromptClient(true, nil, fmt.Errorf("Fake error")),
			},
			expectedErr: false,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newFakeDiscoverClient(nil),
				generateClient: newFakeGenerateClient(customerrors.New(customerrors.ErrorTypeOverwriteConfig, fmt.Errorf("Fake error"))),
				promptClient:   newFakePromptClient(true, nil, fmt.Errorf("Fake error")),
			},
			expectedErr: true,
		},
		{
			menuClient: &MenuClient{
				discoverClient: newFakeDiscoverClient(nil),
				generateClient: newFakeGenerateClient(customerrors.New(customerrors.ErrorTypeOverwriteConfig, fmt.Errorf("Fake error"))),
				promptClient:   newFakePromptClient(false, nil, nil),
			},
			expectedErr: false,
		},
	}

	for idx, c := range cases {
		err := c.menuClient.Menu(ctx)
		if err != nil && !c.expectedErr {
			t.Errorf("Expected err (%d) to be nil: %q", idx, err)
		}
		if err == nil && c.expectedErr {
			t.Errorf("Expected err (%d) not to be nil", idx)
		}
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
		if len(flags) != c.expectedLength {
			t.Errorf("Expected flags length to be '%d' but was: %d", c.expectedLength, len(flags))
		}
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
			if flag.(*cli.StringFlag).Required {
				t.Errorf("Expected flag to be 'false' but was 'true'")
			}
		}
	}
}

type fakeDiscoverClient struct {
	fakeError error
}

func newFakeDiscoverClient(fakeError error) DiscoverInterface {
	return &fakeDiscoverClient{
		fakeError: fakeError,
	}
}

func (client *fakeDiscoverClient) Discover(ctx context.Context) (string, error) {
	return "", nil
}
func (client *fakeDiscoverClient) Run(ctx context.Context) ([]discover, error) {
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

type fakeGenerateClient struct {
	fakeError error
	overwrite bool
}

func newFakeGenerateClient(fakeError error) GenerateInterface {
	return &fakeGenerateClient{
		fakeError: fakeError,
		overwrite: false,
	}
}

func (client *fakeGenerateClient) Generate(ctx context.Context) error {
	if customerrors.To(client.fakeError).ErrorType == customerrors.ErrorTypeOverwriteConfig {
		if !client.overwrite {
			return client.fakeError
		}
		return nil
	}

	return client.fakeError
}

func (client *fakeGenerateClient) Merge(new GenerateClient) {
	if new.overwrite != client.overwrite {
		client.overwrite = new.overwrite
	}
}

type fakePromptClient struct {
	userSelectedOverwrite bool
	selectClusterError    error
	overwriteConfigError  error
}

func newFakePromptClient(userSelectedOverwrite bool, selectClusterError error, overwriteConfigError error) promptInterface {
	return &fakePromptClient{
		userSelectedOverwrite: userSelectedOverwrite,
		selectClusterError:    selectClusterError,
		overwriteConfigError:  overwriteConfigError,
	}
}

func (client *fakePromptClient) selectCluster(apps []discover) (discover, error) {
	if len(apps) == 0 {
		return discover{}, fmt.Errorf("Empty array")
	}
	return apps[0], client.selectClusterError
}

func (client *fakePromptClient) overwriteConfig() (bool, error) {
	return client.userSelectedOverwrite, client.overwriteConfigError
}
