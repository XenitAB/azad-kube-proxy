package actions

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-logr/logr"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestNewDiscoverClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	client := &DiscoverClient{}

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
					ci, err := NewDiscoverClient(ctx, c)
					if err != nil {
						return err
					}

					client = ci.(*DiscoverClient)

					return nil
				},
			},
		},
	}

	cases := []struct {
		cliApp              *cli.App
		args                []string
		expectedConfig      *DiscoverClient
		expectedErrContains string
		outBuffer           bytes.Buffer
		errBuffer           bytes.Buffer
	}{
		{
			cliApp: app,
			args:   []string{"fake-binary", "test"},
			expectedConfig: &DiscoverClient{
				outputType: tableOutputType,
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--output=TABLE"},
			expectedConfig: &DiscoverClient{
				outputType: tableOutputType,
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--output=JSON"},
			expectedConfig: &DiscoverClient{
				outputType: jsonOutputType,
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--output=FAKE"},
			expectedConfig: &DiscoverClient{
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

		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
		} else {
			require.NoError(t, err)
			require.Equal(t, c.expectedConfig.outputType, client.outputType)
		}

		client = &DiscoverClient{}
	}
}

func TestDiscover(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	clientID := getEnvOrSkip(t, "CLIENT_ID")
	clientSecret := getEnvOrSkip(t, "CLIENT_SECRET")
	resource := getEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

	ctx := logr.NewContext(context.Background(), logr.Discard())

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
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Contains(t, rawRes, c.expectedOutputContains)
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
			require.Equal(t, c.expectedOutput[i].ClusterName, d.ClusterName)
			require.Equal(t, c.expectedOutput[i].Resource, d.Resource)
			require.Equal(t, c.expectedOutput[i].ProxyURL, d.ProxyURL)
		}
	}
}

func toStringPtr(s string) *string {
	return &s
}

func toStringArrayPtr(s []string) *[]string {
	return &s
}
