package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	k8sclientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

func TestNewLoginClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	client := &LoginClient{}

	app := &cli.App{
		Name:  "test",
		Usage: "test",
		Commands: []*cli.Command{
			{
				Name:    "test",
				Aliases: []string{"t"},
				Usage:   "test",
				Flags:   LoginFlags(ctx),
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
			if client.clusterName != c.expectedConfig.clusterName {
				t.Errorf("Expected client.clusterName to be '%s' but was: %s", c.expectedConfig.clusterName, client.clusterName)
			}
			if client.resource != c.expectedConfig.resource {
				t.Errorf("Expected client.resource to be '%s' but was: %s", c.expectedConfig.resource, client.resource)
			}
		}
		client = &LoginClient{}
	}
}

func TestLogin(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	clientID := getEnvOrSkip(t, "TEST_USER_SP_CLIENT_ID")
	clientSecret := getEnvOrSkip(t, "TEST_USER_SP_CLIENT_SECRET")
	resource := getEnvOrSkip(t, "TEST_USER_SP_RESOURCE")

	restoreTenantID := tempChangeEnv("AZURE_TENANT_ID", tenantID)
	defer restoreTenantID()

	restoreClientID := tempChangeEnv("AZURE_CLIENT_ID", clientID)
	defer restoreClientID()

	restoreClientSecret := tempChangeEnv("AZURE_CLIENT_SECRET", clientSecret)
	defer restoreClientSecret()

	curDir, err := os.Getwd()
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	tokenCacheFile := fmt.Sprintf("%s/../../../tmp/%s", curDir, tokenCacheFileName)
	defer deleteFile(t, tokenCacheFile)

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
			tokenRes := &k8sclientauth.ExecCredential{}
			err = json.Unmarshal([]byte(rawRes), &tokenRes)
			if err != nil && c.expectedErrContains == "" {
				t.Errorf("Expected err to be nil: %q", err)
			}

			if tokenRes.APIVersion != "client.authentication.k8s.io/v1beta1" {
				t.Errorf("Expected tokenRes.APIVersion to be '%s' but was: %s", "client.authentication.k8s.io/v1beta1", tokenRes.APIVersion)
			}

			if tokenRes.Kind != "ExecCredential" {
				t.Errorf("Expected tokenRes.Kind to be '%s' but was: %s", "ExecCredential", tokenRes.Kind)
			}
		}
	}
}

func TestGetTokenCacheDirectory(t *testing.T) {
	usr, err := user.Current()
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
			expectedResult:  fmt.Sprintf("%s/.kube", usr.HomeDir),
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
