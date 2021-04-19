package actions

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/urfave/cli/v2"
)

func TestNewMenuClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

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
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	clientID := getEnvOrSkip(t, "CLIENT_ID")
	clientSecret := getEnvOrSkip(t, "CLIENT_SECRET")
	resource := getEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	cases := []struct {
		DiscoverClient         *DiscoverClient
		expectedOutputContains string
		expectedErrContains    string
	}{
		{
			DiscoverClient: &DiscoverClient{
				outputType:             tableOutputType,
				tenantID:               tenantID,
				clientID:               clientID,
				clientSecret:           clientSecret,
				enableClientSecretAuth: true,
				enableAzureCliToken:    false,
				enableMsiAuth:          false,
			},
			expectedOutputContains: resource,
			expectedErrContains:    "",
		},
		{
			DiscoverClient: &DiscoverClient{
				outputType:             jsonOutputType,
				tenantID:               tenantID,
				clientID:               clientID,
				clientSecret:           clientSecret,
				enableClientSecretAuth: true,
				enableAzureCliToken:    false,
				enableMsiAuth:          false,
			},
			expectedOutputContains: resource,
			expectedErrContains:    "",
		},
		{
			DiscoverClient: &DiscoverClient{
				outputType:             jsonOutputType,
				tenantID:               tenantID,
				clientID:               clientID,
				clientSecret:           clientSecret,
				enableClientSecretAuth: false,
				enableAzureCliToken:    false,
				enableMsiAuth:          false,
			},
			expectedOutputContains: "",
			expectedErrContains:    "Authentication error: Please validate that you are logged on using the correct credentials",
		},
	}

	for _, c := range cases {
		rawRes, err := c.DiscoverClient.Discover(ctx)
		if err != nil && c.expectedErrContains == "" {
			t.Errorf("Expected err to be nil: %q", err)
		}

		if err == nil && c.expectedErrContains != "" {
			t.Errorf("Expected err to contain '%s' but was nil", c.expectedErrContains)
		}

		if err != nil && c.expectedErrContains != "" {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain '%s' but was: %q", c.expectedErrContains, err)
			}
		}

		if c.expectedErrContains == "" {
			if !strings.Contains(rawRes, c.expectedOutputContains) {
				t.Errorf("Expected output to contain '%s' but was: %s", c.expectedErrContains, rawRes)
			}
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
