package actions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	"github.com/urfave/cli/v2"
	k8sclientauth "k8s.io/client-go/pkg/apis/clientauthentication/v1beta1"
)

func TestNewLoginConfig(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	cfg := LoginConfig{}

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
					var err error
					cfg, err = NewLoginConfig(ctx, c)
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
		expectedConfig      LoginConfig
		expectedErrContains string
		outBuffer           bytes.Buffer
		errBuffer           bytes.Buffer
	}{
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test"},
			expectedConfig:      LoginConfig{},
			expectedErrContains: "cluster-name",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test", "--cluster-name=test"},
			expectedConfig:      LoginConfig{},
			expectedErrContains: "resource",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--cluster-name=test", "--resource=https://fake"},
			expectedConfig: LoginConfig{
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
			if cfg.clusterName != c.expectedConfig.clusterName {
				t.Errorf("Expected cfg.clusterName to be '%s' but was: %s", c.expectedConfig.clusterName, cfg.clusterName)
			}
			if cfg.resource != c.expectedConfig.resource {
				t.Errorf("Expected cfg.resource to be '%s' but was: %s", c.expectedConfig.resource, cfg.resource)
			}
		}
		cfg = LoginConfig{}
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
	tokenCacheFile := fmt.Sprintf("%s/../../../tmp/test-login-token-cache", curDir)
	defer deleteFile(t, tokenCacheFile)

	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	cfg := LoginConfig{
		clusterName: "test",
		resource:    resource,
		tokenCache:  tokenCacheFile,
		defaultAzureCredentialOptions: &azidentity.DefaultAzureCredentialOptions{
			ExcludeAzureCLICredential:    true,
			ExcludeEnvironmentCredential: false,
			ExcludeMSICredential:         true,
		},
	}

	tokenCacheFileErr := fmt.Sprintf("%s/../../../tmp/test-login-token-cache-err", curDir)
	cfgErr := LoginConfig{
		clusterName: "test",
		resource:    resource,
		tokenCache:  tokenCacheFileErr,
		defaultAzureCredentialOptions: &azidentity.DefaultAzureCredentialOptions{
			ExcludeAzureCLICredential:    true,
			ExcludeEnvironmentCredential: true,
			ExcludeMSICredential:         true,
		},
	}

	cases := []struct {
		loginConfig         LoginConfig
		expectedErrContains string
	}{
		{
			loginConfig:         cfg,
			expectedErrContains: "",
		},
		{
			loginConfig:         cfgErr,
			expectedErrContains: "Authentication error:",
		},
	}

	for _, c := range cases {
		rawRes, err := Login(ctx, c.loginConfig)
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
