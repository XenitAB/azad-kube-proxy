package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	k8sclientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

func TestLogin(t *testing.T) {
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	clientID := testGetEnvOrSkip(t, "TEST_USER_SP_CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "TEST_USER_SP_CLIENT_SECRET")
	resource := testGetEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

	restoreTenantID := testTempChangeEnv(t, "AZURE_TENANT_ID", tenantID)
	defer restoreTenantID()

	restoreClientID := testTempChangeEnv(t, "AZURE_CLIENT_ID", clientID)
	defer restoreClientID()

	restoreClientSecret := testTempChangeEnv(t, "AZURE_CLIENT_SECRET", clientSecret)
	defer restoreClientSecret()

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ctx := logr.NewContext(context.Background(), logr.Discard())
	client := &LoginClient{
		clusterName:   "test",
		resource:      resource,
		tokenCacheDir: tmpDir,
		defaultAzureCredentialOptions: defaultAzureCredentialOptions{
			excludeAzureCLICredential:    true,
			excludeEnvironmentCredential: false,
			excludeMSICredential:         true,
		},
	}

	errTmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	clientErr := &LoginClient{
		clusterName:   "test",
		resource:      resource,
		tokenCacheDir: errTmpDir,
		defaultAzureCredentialOptions: defaultAzureCredentialOptions{
			excludeAzureCLICredential:    true,
			excludeEnvironmentCredential: true,
			excludeMSICredential:         true,
		},
	}

	cases := []struct {
		LoginClient         *LoginClient
		expectedErrContains string
	}{
		{
			LoginClient:         client,
			expectedErrContains: "",
		},
		{
			LoginClient:         clientErr,
			expectedErrContains: "Authentication error:",
		},
	}

	for _, c := range cases {
		rawRes, err := c.LoginClient.Login(ctx)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)

		tokenRes := &k8sclientauth.ExecCredential{}
		err = json.Unmarshal([]byte(rawRes), &tokenRes)
		require.NoError(t, err)
		require.Equal(t, "client.authentication.k8s.io/v1beta1", tokenRes.APIVersion)
		require.Equal(t, "ExecCredential", tokenRes.Kind)
	}
}

func TestGetTokenCacheDirectory(t *testing.T) {
	osUserHomeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	cases := []struct {
		testDescription string
		tokenCacheDir   string
		kubeConfig      string
		expectedResult  string
	}{
		{
			testDescription: "all inputs empty strings",
			tokenCacheDir:   "",
			kubeConfig:      "",
			expectedResult:  fmt.Sprintf("%s/.kube", osUserHomeDir),
		},
		{
			testDescription: "kubeConfig set but not tokenCachePath",
			tokenCacheDir:   "",
			kubeConfig:      "/foo/bar/config",
			expectedResult:  "/foo/bar",
		},
		{
			testDescription: "tokenCachePath set but not kubeConfig",
			tokenCacheDir:   "/foo/bar",
			kubeConfig:      "",
			expectedResult:  "/foo/bar",
		},
		{
			testDescription: "tokenCachePath set as well as kubeConfig",
			tokenCacheDir:   "/foo/bar",
			kubeConfig:      "/foo/baz",
			expectedResult:  "/foo/bar",
		},
	}

	for i, c := range cases {
		t.Logf("Test #%d: %s", i, c.testDescription)
		result := getTokenCacheDirectory(c.tokenCacheDir, c.kubeConfig)
		require.Equal(t, filepath.Clean(c.expectedResult), result)
	}
}
