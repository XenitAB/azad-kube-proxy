package main

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
	"github.com/stretchr/testify/require"
)

func TestDiscover(t *testing.T) {
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	clientID := testGetEnvOrSkip(t, "CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "CLIENT_SECRET")
	resource := testGetEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

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
					DisplayName:    testToPtr(t, "fake"),
					IdentifierUris: testToPtr(t, []string{"https://fake"}),
					Tags:           testToPtr(t, []string{"azad-kube-proxy"}),
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
					DisplayName:    testToPtr(t, "fake"),
					IdentifierUris: testToPtr(t, []string{"https://fake"}),
					Tags:           testToPtr(t, []string{"azad-kube-proxy"}),
				},
				{
					DisplayName:    testToPtr(t, "fake2"),
					IdentifierUris: testToPtr(t, []string{"https://fake2"}),
					Tags:           testToPtr(t, []string{"azad-kube-proxy"}),
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
					DisplayName:    testToPtr(t, "fake"),
					IdentifierUris: testToPtr(t, []string{"https://fake"}),
					Tags:           testToPtr(t, []string{"azad-kube-proxy", "cluster_name:newfake"}),
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
					DisplayName:    testToPtr(t, "fake"),
					IdentifierUris: testToPtr(t, []string{"https://fake"}),
					Tags:           testToPtr(t, []string{"azad-kube-proxy", "proxy_url:https://newfake"}),
				},
				{
					DisplayName:    testToPtr(t, "fake"),
					IdentifierUris: testToPtr(t, []string{"https://fake"}),
					Tags:           testToPtr(t, []string{"azad-kube-proxy", "cluster_name:newfake2", "proxy_url:https://newfake2"}),
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
					DisplayName:    testToPtr(t, "fake"),
					IdentifierUris: testToPtr(t, []string{"https://fake"}),
					Tags:           testToPtr(t, []string{"azad-kube-proxy", "fake"}),
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

func testToPtr[T any](t *testing.T, s T) *T {
	t.Helper()

	return &s
}
