package actions

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
	"github.com/urfave/cli/v2"
	k8sclientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

func TestNewLoginClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	client := &LoginClient{}

	loginFlags, err := LoginFlags(ctx)
	require.NoError(t, err)

	app := &cli.App{
		Name:  "test",
		Usage: "test",
		Commands: []*cli.Command{
			{
				Name:    "test",
				Aliases: []string{"t"},
				Usage:   "test",
				Flags:   loginFlags,
				Action: func(c *cli.Context) error {
					ci, err := NewLoginClient(ctx, c)
					if err != nil {
						return err
					}

					client = ci.(*LoginClient)

					return nil
				},
			},
		},
	}

	cases := []struct {
		cliApp              *cli.App
		args                []string
		expectedConfig      *LoginClient
		expectedErrContains string
		outBuffer           bytes.Buffer
		errBuffer           bytes.Buffer
	}{
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test"},
			expectedConfig:      &LoginClient{},
			expectedErrContains: "cluster-name",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test", "--cluster-name=test"},
			expectedConfig:      &LoginClient{},
			expectedErrContains: "resource",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--cluster-name=test", "--resource=https://fake"},
			expectedConfig: &LoginClient{
				clusterName: "test",
				resource:    "https://fake",
			},
			expectedErrContains: "",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
	}

	for _, c := range cases {
		client = &LoginClient{}
		c.cliApp.Writer = &c.outBuffer
		c.cliApp.ErrWriter = &c.errBuffer
		err := c.cliApp.Run(c.args)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.expectedConfig.clusterName, client.clusterName)
		require.Equal(t, c.expectedConfig.resource, client.resource)

	}
}

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

	curDir, err := os.Getwd()
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	tokenCacheFile := fmt.Sprintf("%s/../../../tmp/%s", curDir, tokenCacheFileName)
	defer testDeleteFile(t, tokenCacheFile)

	ctx := logr.NewContext(context.Background(), logr.Discard())
	client := &LoginClient{
		clusterName:   "test",
		resource:      resource,
		tokenCacheDir: filepath.Dir(tokenCacheFile),
		defaultAzureCredentialOptions: defaultAzureCredentialOptions{
			excludeAzureCLICredential:    true,
			excludeEnvironmentCredential: false,
			excludeMSICredential:         true,
		},
	}

	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	clientErr := &LoginClient{
		clusterName:   "test",
		resource:      resource,
		tokenCacheDir: tmpDir,
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
		require.Equal(t, c.expectedResult, result)
	}
}
