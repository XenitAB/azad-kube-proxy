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

func TestNewGenerateClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	client := &GenerateClient{}

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
					ci, err := NewGenerateClient(ctx, c)
					if err != nil {
						return err
					}

					client = ci.(*GenerateClient)

					return nil
				},
			},
		},
	}

	cases := []struct {
		cliApp              *cli.App
		args                []string
		expectedConfig      *GenerateClient
		expectedErrContains string
		outBuffer           bytes.Buffer
		errBuffer           bytes.Buffer
	}{
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test"},
			expectedConfig:      &GenerateClient{},
			expectedErrContains: "cluster-name",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test", "--cluster-name=test"},
			expectedConfig:      &GenerateClient{},
			expectedErrContains: "proxy-url",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp:              app,
			args:                []string{"fake-binary", "test", "--cluster-name=test", "--proxy-url=https://fake"},
			expectedConfig:      &GenerateClient{},
			expectedErrContains: "resource",
			outBuffer:           bytes.Buffer{},
			errBuffer:           bytes.Buffer{},
		},
		{
			cliApp: app,
			args:   []string{"fake-binary", "test", "--cluster-name=test", "--proxy-url=https://fake", "--resource=https://fake"},
			expectedConfig: &GenerateClient{
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
			if client.clusterName != c.expectedConfig.clusterName {
				t.Errorf("Expected client.clusterName to be '%s' but was: %s", c.expectedConfig.clusterName, client.clusterName)
			}
			if client.proxyURL.Host != c.expectedConfig.proxyURL.Host {
				t.Errorf("Expected client.proxyURL.Host to be '%s' but was: %s", c.expectedConfig.proxyURL.Host, client.proxyURL.Host)
			}
			if client.proxyURL.Scheme != c.expectedConfig.proxyURL.Scheme {
				t.Errorf("Expected client.proxyURL.Scheme to be '%s' but was: %s", c.expectedConfig.proxyURL.Scheme, client.proxyURL.Scheme)
			}
			if client.resource != c.expectedConfig.resource {
				t.Errorf("Expected client.resource to be '%s' but was: %s", c.expectedConfig.resource, client.resource)
			}
		}
		client = &GenerateClient{}
	}
}

func TestMergeGenerateClient(t *testing.T) {
	client := &GenerateClient{
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

	client.Merge(GenerateClient{
		clusterName:        "test2",
		proxyURL:           getURL("https://www.example.com"),
		resource:           "https://fake2",
		kubeConfig:         "/tmp2/kubeconfig",
		tokenCache:         "/tmp2/tokencache",
		overwrite:          true,
		insecureSkipVerify: true,
	})

	if client.clusterName != "test2" {
		t.Errorf("Expected client.clusterName to be 'test2' but was: %s", client.clusterName)
	}
	if client.proxyURL.String() != "https://www.example.com" {
		t.Errorf("Expected client.proxyURL.String() to be 'https://www.example.com' but was: %s", client.proxyURL.String())
	}
	if client.resource != "https://fake2" {
		t.Errorf("Expected client.resource to be 'https://fake2' but was: %s", client.resource)
	}
	if client.kubeConfig != "/tmp2/kubeconfig" {
		t.Errorf("Expected client.kubeConfig to be '/tmp2/kubeconfig' but was: %s", client.kubeConfig)
	}
	if client.tokenCache != "/tmp2/tokencache" {
		t.Errorf("Expected client.tokenCache to be '/tmp2/tokencache' but was: %s", client.tokenCache)
	}
	if client.overwrite != true {
		t.Errorf("Expected client.overwrite to be 'true' but was: %t", client.overwrite)
	}
	if client.insecureSkipVerify != true {
		t.Errorf("Expected client.insecureSkipVerify to be 'true' but was: %t", client.insecureSkipVerify)
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

	client := &GenerateClient{
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
				client.proxyURL = getURL("https://localhost:12345")
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
			kubeCfg, err := k8sclientcmd.LoadFromFile(c.GenerateClient.kubeConfig)
			if err != nil && c.expectedErrContains == "" {
				t.Errorf("Expected err to be nil: %q", err)
			}

			if kubeCfg.Clusters[c.GenerateClient.clusterName].Server != fmt.Sprintf("%s://%s", c.GenerateClient.proxyURL.Scheme, c.GenerateClient.proxyURL.Host) {
				t.Errorf("Expected kubeCfg.Clusters[c.GenerateClient.clusterName].Server to be '%s' but was: %s", fmt.Sprintf("%s://%s", c.GenerateClient.proxyURL.Scheme, c.GenerateClient.proxyURL.Host), kubeCfg.Clusters[c.GenerateClient.clusterName].Server)
			}

			if len(kubeCfg.Clusters[c.GenerateClient.clusterName].CertificateAuthorityData) == 0 {
				t.Errorf("Expected length of kubeCfg.Clusters[c.GenerateClient.clusterName].CertificateAuthorityData to be lager than 0")
			}
		}
	}
}

func getURL(s string) url.URL {
	res, _ := url.Parse(s)
	return *res
}
