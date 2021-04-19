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
	"github.com/urfave/cli/v2"
	k8sclientcmd "k8s.io/client-go/tools/clientcmd"
)

func TestNewGenerateConfig(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
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

func TestMergeGenerateConfig(t *testing.T) {
	cfg := &GenerateConfig{
		clusterName:        "test",
		proxyURL:           getURL("https://www.google.com"),
		resource:           "https://fake",
		kubeConfig:         "/tmp/kubeconfig",
		tokenCache:         "/tmp/tokencache",
		overwrite:          false,
		insecureSkipVerify: false,
		defaultAzureCredentialOptions: &azidentity.DefaultAzureCredentialOptions{
			ExcludeAzureCLICredential:    false,
			ExcludeEnvironmentCredential: false,
			ExcludeMSICredential:         false,
		},
	}

	cfg.Merge(GenerateConfig{
		clusterName:        "test2",
		proxyURL:           getURL("https://www.example.com"),
		resource:           "https://fake2",
		kubeConfig:         "/tmp2/kubeconfig",
		tokenCache:         "/tmp2/tokencache",
		overwrite:          true,
		insecureSkipVerify: true,
	})

	if cfg.clusterName != "test2" {
		t.Errorf("Expected cfg.clusterName to be 'test2' but was: %s", cfg.clusterName)
	}
	if cfg.proxyURL.String() != "https://www.example.com" {
		t.Errorf("Expected cfg.proxyURL.String() to be 'https://www.example.com' but was: %s", cfg.proxyURL.String())
	}
	if cfg.resource != "https://fake2" {
		t.Errorf("Expected cfg.resource to be 'https://fake2' but was: %s", cfg.resource)
	}
	if cfg.kubeConfig != "/tmp2/kubeconfig" {
		t.Errorf("Expected cfg.kubeConfig to be '/tmp2/kubeconfig' but was: %s", cfg.kubeConfig)
	}
	if cfg.tokenCache != "/tmp2/tokencache" {
		t.Errorf("Expected cfg.tokenCache to be '/tmp2/tokencache' but was: %s", cfg.tokenCache)
	}
	if cfg.overwrite != true {
		t.Errorf("Expected cfg.overwrite to be 'true' but was: %t", cfg.overwrite)
	}
	if cfg.insecureSkipVerify != true {
		t.Errorf("Expected cfg.insecureSkipVerify to be 'true' but was: %t", cfg.insecureSkipVerify)
	}
}

func TestGenerate(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
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
			expectedErrContains: "Overwrite config error:",
		},
		{
			generateConfig: cfg,
			generateConfigFunc: func(oldCfg GenerateConfig) GenerateConfig {
				cfg := oldCfg
				cfg.proxyURL = getURL("https://localhost:12345")
				cfg.overwrite = true
				return cfg
			},
			expectedErrContains: "CA certificate error:",
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
