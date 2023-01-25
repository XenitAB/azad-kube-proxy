package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

func TestRunGenerate(t *testing.T) {
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

	cfg := generateConfig{
		ClusterName:           "ze-cluster",
		ProxyURL:              srv.URL,
		Resource:              "ze-resource",
		KubeConfig:            kubeConfigFile,
		TokenCacheDir:         tokenCacheDir,
		Overwrite:             false,
		TLSInsecureSkipVerify: true,
	}
	authCfg := authConfig{
		excludeAzureCLIAuth:    false,
		excludeEnvironmentAuth: true,
		excludeMSIAuth:         true,
	}
	err = runGenerate(ctx, cfg, authCfg)
	require.NoError(t, err)

	require.ErrorContains(t, runGenerate(ctx, generateConfig{}, authConfig{}), "Unable to load file")
	require.ErrorContains(t, runGenerate(ctx, generateConfig{ProxyURL: "$#%~!-_"}, authConfig{}), "invalid URL escape")
}

func TestGenerate(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "test response")
	}))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tokenCacheDir := tmpDir
	kubeConfigFile := fmt.Sprintf("%s/kubeconfig", tmpDir)

	client := &GenerateClient{
		clusterName:        "test",
		proxyURL:           testGetURL(t, ts.URL),
		resource:           "https://fake",
		kubeConfig:         kubeConfigFile,
		tokenCacheDir:      tokenCacheDir,
		overwrite:          false,
		insecureSkipVerify: true,
		defaultAzureCredentialOptions: defaultAzureCredentialOptions{
			excludeAzureCLICredential:    false,
			excludeEnvironmentCredential: false,
			excludeMSICredential:         false,
		},
	}

	cases := []struct {
		testDescription     string
		generateClient      *GenerateClient
		generateClientFunc  func(oldCfg *GenerateClient) *GenerateClient
		expectedErrContains string
	}{
		{
			testDescription:     "plain",
			generateClient:      client,
			expectedErrContains: "",
		},
		{
			testDescription:     "config error",
			generateClient:      client,
			expectedErrContains: "Overwrite config error:",
		},
		{
			testDescription: "ca error",
			generateClient:  client,
			generateClientFunc: func(oldClient *GenerateClient) *GenerateClient {
				client := oldClient
				client.proxyURL = testGetURL(t, "https://localhost:12345")
				client.overwrite = true
				return client
			},
			expectedErrContains: "CA certificate error:",
		},
	}

	for i, c := range cases {
		t.Logf("Test #%d: %s", i, c.testDescription)
		if c.generateClientFunc != nil {
			c.generateClient = c.generateClientFunc(c.generateClient)
		}

		err := c.generateClient.Generate(ctx)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)

		kubeCfg, err := k8sclientcmd.LoadFromFile(c.generateClient.kubeConfig)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("%s://%s", c.generateClient.proxyURL.Scheme, c.generateClient.proxyURL.Host), kubeCfg.Clusters[c.generateClient.clusterName].Server)
		require.NotEmpty(t, kubeCfg.Clusters[c.generateClient.clusterName].CertificateAuthorityData)

	}
}

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

func testGetURL(t *testing.T, s string) url.URL {
	t.Helper()

	res, err := url.Parse(s)
	require.NoError(t, err)

	return *res
}
