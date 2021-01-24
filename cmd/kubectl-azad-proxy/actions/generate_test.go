package actions

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	logrTesting "github.com/go-logr/logr/testing"
	"github.com/urfave/cli/v2"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

func TestNewGenerateConfig(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})
	cfg := GenerateConfig{}

	app := &cli.App{
		Name:  "test",
		Usage: "test",
		Commands: []*cli.Command{
			{
				Name:    "test",
				Aliases: []string{"t"},
				Usage:   "test",
				Flags:   GenerateFlags(ctx),
				Action: func(c *cli.Context) error {
					var err error
					cfg, err = NewGenerateConfig(ctx, c)
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
		expectedConfig      GenerateConfig
		expectedErrContains string
		outBuffer           bytes.Buffer
		errBuffer           bytes.Buffer
	}{
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test"},
			expectedConfig:      GenerateConfig{},
			expectedErrContains: "cluster-name",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test", "--cluster-name=test"},
			expectedConfig:      GenerateConfig{},
			expectedErrContains: "proxy-url",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test", "--cluster-name=test", "--proxy-url=https://fake"},
			expectedConfig:      GenerateConfig{},
			expectedErrContains: "resource",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--cluster-name=test", "--proxy-url=https://fake", "--resource=https://fake"},
			expectedConfig: GenerateConfig{
				clusterName: "test",
				proxyURL:    getURL("https://fake"),
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
			if cfg.proxyURL.Host != c.expectedConfig.proxyURL.Host {
				t.Errorf("Expected cfg.proxyURL.Host to be '%s' but was: %s", c.expectedConfig.proxyURL.Host, cfg.proxyURL.Host)
			}
			if cfg.proxyURL.Scheme != c.expectedConfig.proxyURL.Scheme {
				t.Errorf("Expected cfg.proxyURL.Scheme to be '%s' but was: %s", c.expectedConfig.proxyURL.Scheme, cfg.proxyURL.Scheme)
			}
			if cfg.resource != c.expectedConfig.resource {
				t.Errorf("Expected cfg.resource to be '%s' but was: %s", c.expectedConfig.resource, cfg.resource)
			}
		}
		cfg = GenerateConfig{}
	}
}

func TestGenerate(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})
	curDir, err := os.Getwd()
	if err != nil {
		t.Errorf("Expected err to be nil: %q", err)
	}
	tokenCacheFile := fmt.Sprintf("%s/../../../tmp/test-cached-tokens-generate", curDir)
	kubeConfigFile := fmt.Sprintf("%s/../../../tmp/test-generate-kubeconfig", curDir)
	defer deleteFile(t, kubeConfigFile)

	cfg := GenerateConfig{
		clusterName:        "test",
		proxyURL:           getURL("https://www.google.com"),
		resource:           "https://fake",
		kubeConfig:         kubeConfigFile,
		tokenCache:         tokenCacheFile,
		overwrite:          false,
		insecureSkipVerify: false,
		defaultAzureCredentialOptions: &azidentity.DefaultAzureCredentialOptions{
			ExcludeAzureCLICredential:    false,
			ExcludeEnvironmentCredential: false,
			ExcludeMSICredential:         false,
		},
	}

	cases := []struct {
		generateConfig      GenerateConfig
		generateConfigFunc  func(oldCfg GenerateConfig) GenerateConfig
		expectedErrContains string
	}{
		{
			generateConfig:      cfg,
			expectedErrContains: "",
		},
		{
			generateConfig:      cfg,
			expectedErrContains: "but overwrite is false",
		},
		{
			generateConfig: cfg,
			generateConfigFunc: func(oldCfg GenerateConfig) GenerateConfig {
				cfg := oldCfg
				cfg.proxyURL = getURL("https://localhost:12345")
				cfg.overwrite = true
				return cfg
			},
			expectedErrContains: "connect: connection refused",
		},
	}

	for _, c := range cases {
		if c.generateConfigFunc != nil {
			c.generateConfig = c.generateConfigFunc(c.generateConfig)
		}

		err := Generate(ctx, c.generateConfig)
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
			kubeCfg, err := k8sclientcmd.LoadFromFile(c.generateConfig.kubeConfig)
			if err != nil && c.expectedErrContains == "" {
				t.Errorf("Expected err to be nil: %q", err)
			}

			if kubeCfg.Clusters[c.generateConfig.clusterName].Server != fmt.Sprintf("%s://%s", c.generateConfig.proxyURL.Scheme, c.generateConfig.proxyURL.Host) {
				t.Errorf("Expected kubeCfg.Clusters[c.generateConfig.clusterName].Server to be '%s' but was: %s", fmt.Sprintf("%s://%s", c.generateConfig.proxyURL.Scheme, c.generateConfig.proxyURL.Host), kubeCfg.Clusters[c.generateConfig.clusterName].Server)
			}

			if len(kubeCfg.Clusters[c.generateConfig.clusterName].CertificateAuthorityData) < 0 {
				t.Errorf("Expected length of kubeCfg.Clusters[c.generateConfig.clusterName].CertificateAuthorityData to be lager than 0 but was: %d", len(kubeCfg.Clusters[c.generateConfig.clusterName].CertificateAuthorityData))
			}
		}
	}
}

func getURL(s string) url.URL {
	res, _ := url.Parse(s)
	return *res
}
