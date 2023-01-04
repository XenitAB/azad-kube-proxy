package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

func TestMergeGenerateClient(t *testing.T) {
	client := &GenerateClient{
		clusterName:        "test",
		proxyURL:           testGetURL(t, "https://www.google.com"),
		resource:           "https://fake",
		kubeConfig:         "/tmp/kubeconfig",
		tokenCacheDir:      "/tmp/tokencache",
		overwrite:          false,
		insecureSkipVerify: false,
		defaultAzureCredentialOptions: defaultAzureCredentialOptions{
			excludeAzureCLICredential:    false,
			excludeEnvironmentCredential: false,
			excludeMSICredential:         false,
		},
	}

	client.Merge(GenerateClient{
		clusterName:        "test2",
		proxyURL:           testGetURL(t, "https://www.example.com"),
		resource:           "https://fake2",
		kubeConfig:         "/tmp2/kubeconfig",
		tokenCacheDir:      "/tmp2/tokencache",
		overwrite:          true,
		insecureSkipVerify: true,
	})

	require.Equal(t, "test2", client.clusterName)
	require.Equal(t, "https://www.example.com", client.proxyURL.String())
	require.Equal(t, "https://fake2", client.resource)
	require.Equal(t, "/tmp2/kubeconfig", client.kubeConfig)
	require.Equal(t, "/tmp2/tokencache", client.tokenCacheDir)
	require.Equal(t, true, client.overwrite)
	require.Equal(t, true, client.insecureSkipVerify)
}

func TestGenerate(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	tokenCacheDir := tmpDir
	kubeConfigFile := fmt.Sprintf("%s/kubeconfig", tmpDir)
	defer testDeleteFile(t, kubeConfigFile)

	client := &GenerateClient{
		clusterName:        "test",
		proxyURL:           testGetURL(t, "https://www.google.com"),
		resource:           "https://fake",
		kubeConfig:         kubeConfigFile,
		tokenCacheDir:      tokenCacheDir,
		overwrite:          false,
		insecureSkipVerify: false,
		defaultAzureCredentialOptions: defaultAzureCredentialOptions{
			excludeAzureCLICredential:    false,
			excludeEnvironmentCredential: false,
			excludeMSICredential:         false,
		},
	}

	cases := []struct {
		GenerateClient      *GenerateClient
		GenerateClientFunc  func(oldCfg *GenerateClient) *GenerateClient
		expectedErrContains string
	}{
		{
			GenerateClient:      client,
			expectedErrContains: "",
		},
		{
			GenerateClient:      client,
			expectedErrContains: "Overwrite config error:",
		},
		{
			GenerateClient: client,
			GenerateClientFunc: func(oldClient *GenerateClient) *GenerateClient {
				client := oldClient
				client.proxyURL = testGetURL(t, "https://localhost:12345")
				client.overwrite = true
				return client
			},
			expectedErrContains: "CA certificate error:",
		},
	}

	for _, c := range cases {
		if c.GenerateClientFunc != nil {
			c.GenerateClient = c.GenerateClientFunc(c.GenerateClient)
		}

		err := c.GenerateClient.Generate(ctx)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)

		kubeCfg, err := k8sclientcmd.LoadFromFile(c.GenerateClient.kubeConfig)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%s://%s", c.GenerateClient.proxyURL.Scheme, c.GenerateClient.proxyURL.Host), kubeCfg.Clusters[c.GenerateClient.clusterName].Server)
		require.NotEmpty(t, kubeCfg.Clusters[c.GenerateClient.clusterName].CertificateAuthorityData)

	}
}

func testGetURL(t *testing.T, s string) url.URL {
	t.Helper()

	res, err := url.Parse(s)
	require.NoError(t, err)

	return *res
}
