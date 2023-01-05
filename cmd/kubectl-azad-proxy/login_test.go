package main

import (
	"bytes"
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

func TestRunLogin(t *testing.T) {
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

	ctx := logr.NewContext(context.Background(), logr.Discard())

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	errTmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(errTmpDir)

	cases := []struct {
		cfg                 loginConfig
		authCfg             authConfig
		expectedErrContains string
	}{
		{
			cfg: loginConfig{
				ClusterName:   "test",
				Resource:      resource,
				TokenCacheDir: tmpDir,
			},
			authCfg: authConfig{

				excludeAzureCLIAuth:    true,
				excludeEnvironmentAuth: false,
				excludeMSIAuth:         true,
			},
			expectedErrContains: "",
		},
		{
			cfg: loginConfig{
				ClusterName:   "test",
				Resource:      resource,
				TokenCacheDir: errTmpDir,
			},
			authCfg: authConfig{
				excludeAzureCLIAuth:    true,
				excludeEnvironmentAuth: true,
				excludeMSIAuth:         true,
			},
			expectedErrContains: "Authentication error:",
		},
	}

	for _, c := range cases {
		buffer := &bytes.Buffer{}
		err := runLogin(ctx, buffer, c.cfg, c.authCfg)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)

		tokenRes := &k8sclientauth.ExecCredential{}
		err = json.Unmarshal([]byte(buffer.Bytes()), &tokenRes)
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
