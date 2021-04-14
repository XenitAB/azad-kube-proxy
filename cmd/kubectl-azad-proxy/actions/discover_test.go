package actions

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
	"github.com/urfave/cli/v2"
)

func TestNewDiscoverConfig(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	cfg := DiscoverConfig{}

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
				Flags:   DiscoverFlags(ctx),
				Action: func(c *cli.Context) error {
					var err error
					cfg, err = NewDiscoverConfig(ctx, c)
					if err != nil {
						return err
					}
					return nil
				},
			},
		},
	}

	cases := []struct {
		cliApp              *cli.App
		args                []string
		expectedConfig      DiscoverConfig
		expectedErrContains string
		outBuffer           bytes.Buffer
		errBuffer           bytes.Buffer
	}{
		{
			cliApp: app,
			args:   []string{"fake-binary", "test"},
			expectedConfig: DiscoverConfig{
				outputType: tableOutputType,
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--output=TABLE"},
			expectedConfig: DiscoverConfig{
				outputType: tableOutputType,
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--output=JSON"},
			expectedConfig: DiscoverConfig{
				outputType: jsonOutputType,
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--output=FAKE"},
			expectedConfig: DiscoverConfig{
				outputType: jsonOutputType,
			},
			expectedErrContains: "Supported outputs are TABLE and JSON. The following was used: FAKE",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
	}

	for _, c := range cases {
		c.cliApp.Writer = &c.outBuffer
		c.cliApp.ErrWriter = &c.errBuffer
		err := c.cliApp.Run(c.args)
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
			if cfg.outputType != c.expectedConfig.outputType {
				t.Errorf("Expected cfg.clusterName to be '%s' but was: %s", c.expectedConfig.outputType, cfg.outputType)
			}
		}

		cfg = DiscoverConfig{}
	}
}

func TestDiscover(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	clientID := getEnvOrSkip(t, "CLIENT_ID")
	clientSecret := getEnvOrSkip(t, "CLIENT_SECRET")
	resource := getEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})

	cases := []struct {
		discoverConfig         DiscoverConfig
		expectedOutputContains string
		expectedErrContains    string
	}{
		{
			discoverConfig: DiscoverConfig{
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
			discoverConfig: DiscoverConfig{
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
			discoverConfig: DiscoverConfig{
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
		rawRes, err := Discover(ctx, c.discoverConfig)
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

func TestGetDiscoverData(t *testing.T) {
	cases := []struct {
		clusterApps    []hamiltonMsgraph.Application
		expectedOutput []discover
	}{
		{
			clusterApps: []hamiltonMsgraph.Application{
				{
					DisplayName:    toStringPtr("fake"),
					IdentifierUris: toStringArrayPtr([]string{"https://fake"}),
					Tags:           toStringArrayPtr([]string{"azad-kube-proxy"}),
				},
			},
			expectedOutput: []discover{
				{
					ClusterName: "fake",
					Resource:    "https://fake",
					ProxyURL:    "https://fake",
				},
			},
		},
		{
			clusterApps: []hamiltonMsgraph.Application{
				{
					DisplayName:    toStringPtr("fake"),
					IdentifierUris: toStringArrayPtr([]string{"https://fake"}),
					Tags:           toStringArrayPtr([]string{"azad-kube-proxy"}),
				},
				{
					DisplayName:    toStringPtr("fake2"),
					IdentifierUris: toStringArrayPtr([]string{"https://fake2"}),
					Tags:           toStringArrayPtr([]string{"azad-kube-proxy"}),
				},
			},
			expectedOutput: []discover{
				{
					ClusterName: "fake",
					Resource:    "https://fake",
					ProxyURL:    "https://fake",
				},
				{
					ClusterName: "fake2",
					Resource:    "https://fake2",
					ProxyURL:    "https://fake2",
				},
			},
		},
		{
			clusterApps: []hamiltonMsgraph.Application{
				{
					DisplayName:    toStringPtr("fake"),
					IdentifierUris: toStringArrayPtr([]string{"https://fake"}),
					Tags:           toStringArrayPtr([]string{"azad-kube-proxy", "cluster_name:newfake"}),
				},
			},
			expectedOutput: []discover{
				{
					ClusterName: "newfake",
					Resource:    "https://fake",
					ProxyURL:    "https://fake",
				},
			},
		},
		{
			clusterApps: []hamiltonMsgraph.Application{
				{
					DisplayName:    toStringPtr("fake"),
					IdentifierUris: toStringArrayPtr([]string{"https://fake"}),
					Tags:           toStringArrayPtr([]string{"azad-kube-proxy", "proxy_url:https://newfake"}),
				},
				{
					DisplayName:    toStringPtr("fake"),
					IdentifierUris: toStringArrayPtr([]string{"https://fake"}),
					Tags:           toStringArrayPtr([]string{"azad-kube-proxy", "cluster_name:newfake2", "proxy_url:https://newfake2"}),
				},
			},
			expectedOutput: []discover{
				{
					ClusterName: "fake",
					Resource:    "https://fake",
					ProxyURL:    "https://newfake",
				},
				{
					ClusterName: "newfake2",
					Resource:    "https://fake",
					ProxyURL:    "https://newfake2",
				},
			},
		},
		{
			clusterApps: []hamiltonMsgraph.Application{
				{
					DisplayName:    toStringPtr("fake"),
					IdentifierUris: toStringArrayPtr([]string{"https://fake"}),
					Tags:           toStringArrayPtr([]string{"azad-kube-proxy", "fake"}),
				},
			},
			expectedOutput: []discover{
				{
					ClusterName: "fake",
					Resource:    "https://fake",
					ProxyURL:    "https://fake",
				},
			},
		},
	}

	for _, c := range cases {
		discoverData := getDiscoverData(c.clusterApps)
		for i, d := range discoverData {
			if c.expectedOutput[i].ClusterName != d.ClusterName {
				t.Errorf("Expected output cluster name to be '%s' but was: %s", c.expectedOutput[i].ClusterName, d.ClusterName)
			}
			if c.expectedOutput[i].Resource != d.Resource {
				t.Errorf("Expected output resource to be '%s' but was: %s", c.expectedOutput[i].Resource, d.Resource)
			}
			if c.expectedOutput[i].ProxyURL != d.ProxyURL {
				t.Errorf("Expected output proxy url to be '%s' but was: %s", c.expectedOutput[i].ProxyURL, d.ProxyURL)
			}
		}
	}
}

func toStringPtr(s string) *string {
	return &s
}

func toStringArrayPtr(s []string) *[]string {
	return &s
}
