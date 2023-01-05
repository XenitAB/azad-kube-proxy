package main

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
)

type discoverConfig struct {
	Output            string `arg:"--output,env:OUTPUT" default:"TABLE" help:"How to output the data"`
	AzureTenantID     string `arg:"--tenant-id,env:AZURE_TENANT_ID" help:"Azure Tenant ID used with ENV auth"`
	AzureClientID     string `arg:"--client-id,env:AZURE_CLIENT_ID" help:"Azure Client ID used with ENV auth"`
	AzureClientSecret string `arg:"--client-secret,env:AZURE_CLIENT_SECRET" help:"Azure Client Secret used with ENV auth"`
}

type generateConfig struct {
	ClusterName           string `arg:"--cluster-name,env:CLUSTER_NAME,required" help:"The name of the Kubernetes cluster / context"`
	ProxyURL              string `arg:"--proxy-url,env:PROXY_URL,required" help:"The proxy url for azad-kube-proxy"`
	Resource              string `arg:"--resource,env:RESOURCE,required" help:"The Azure AD App URI / resource"`
	KubeConfig            string `arg:"--kubeconfig,env:KUBECONFIG" help:"The path of the Kubernetes Config"` // FIXME: Default to fmt.Sprintf("%s/.kube/config", osUserHomeDir)
	TokenCacheDir         string `arg:"--token-cache-dir,env:TOKEN_CACHE_DIR" help:"The directory to where the tokens are cached, defaults to the same as KUBECONFIG"`
	Overwrite             bool   `arg:"--overwrite,env:OVERWRITE_KUBECONFIG" default:"false" help:"If the cluster already exists in the kubeconfig, should it be overwritten?"`
	TLSInsecureSkipVerify bool   `arg:"--tls-insecure-skip-verify,env:TLS_INSECURE_SKIP_VERIFY" default:"false" help:"Should the proxy url server certificate validation be skipped?"`
}

type loginConfig struct {
	ClusterName   string `arg:"--cluster-name,env:CLUSTER_NAME,required" help:"The name of the Kubernetes cluster / context"`
	Resource      string `arg:"--resource,env:RESOURCE,required" help:"The Azure AD App URI / resource"`
	KubeConfig    string `arg:"--kubeconfig,env:KUBECONFIG" help:"The path of the Kubernetes Config"` // FIXME: Default to fmt.Sprintf("%s/.kube/config", osUserHomeDir)
	TokenCacheDir string `arg:"--token-cache-dir,env:TOKEN_CACHE_DIR" help:"The directory to where the tokens are cached, defaults to the same as KUBECONFIG"`
}

type menuConfig struct {
	Output                string `arg:"--output,env:OUTPUT" default:"TABLE" help:"How to output the data"`
	AzureTenantID         string `arg:"--tenant-id,env:AZURE_TENANT_ID" help:"Azure Tenant ID used with ENV auth"`
	AzureClientID         string `arg:"--client-id,env:AZURE_CLIENT_ID" help:"Azure Client ID used with ENV auth"`
	AzureClientSecret     string `arg:"--client-secret,env:AZURE_CLIENT_SECRET" help:"Azure Client Secret used with ENV auth"`
	ClusterName           string `arg:"--cluster-name,env:CLUSTER_NAME" help:"The name of the Kubernetes cluster / context"`
	ProxyURL              string `arg:"--proxy-url,env:PROXY_URL" help:"The proxy url for azad-kube-proxy"`
	Resource              string `arg:"--resource,env:RESOURCE" help:"The Azure AD App URI / resource"`
	KubeConfig            string `arg:"--kubeconfig,env:KUBECONFIG" help:"The path of the Kubernetes Config"` // FIXME: Default to fmt.Sprintf("%s/.kube/config", osUserHomeDir)
	TokenCacheDir         string `arg:"--token-cache-dir,env:TOKEN_CACHE_DIR" help:"The directory to where the tokens are cached, defaults to the same as KUBECONFIG"`
	Overwrite             bool   `arg:"--overwrite,env:OVERWRITE_KUBECONFIG" default:"false" help:"If the cluster already exists in the kubeconfig, should it be overwritten?"`
	TLSInsecureSkipVerify bool   `arg:"--tls-insecure-skip-verify,env:TLS_INSECURE_SKIP_VERIFY" default:"false" help:"Should the proxy url server certificate validation be skipped?"`
}

type authConfig struct {
	excludeAzureCLIAuth    bool
	excludeEnvironmentAuth bool
	excludeMSIAuth         bool
}

type config struct {
	Discover *discoverConfig `arg:"subcommand:discover"`
	Generate *generateConfig `arg:"subcommand:generate"`
	Login    *loginConfig    `arg:"subcommand:login"`
	Menu     *menuConfig     `arg:"subcommand:menu"`

	// Global flags
	Debug                  bool `arg:"--debug" default:"false" help:"Enable debug output"`
	ExcludeAzureCLIAuth    bool `arg:"--exclude-azure-cli-auth,env:EXCLUDE_AZURE_CLI_AUTH" default:"false" help:"Should Azure CLI be excluded from the authentication?"`
	ExcludeEnvironmentAuth bool `arg:"--exclude-environment-auth,env:EXCLUDE_ENVIRONMENT_AUTH" default:"true" help:"Should environment be excluded from the authentication?"`
	ExcludeMSIAuth         bool `arg:"--exclude-msi-auth,env:EXCLUDE_MSI_AUTH" default:"true" help:"Should MSI be excluded from the authentication?"`

	authConfig authConfig
}

func (config) Version() string {
	return fmt.Sprintf("version=%s revision=%s created=%s\n", Version, Revision, Created)
}

func newConfig(args []string) (config, error) {
	cfg := config{}
	parser, err := arg.NewParser(arg.Config{
		Program:   "azad-proxy kubectl plugin",
		IgnoreEnv: false,
	}, &cfg)
	if err != nil {
		return config{}, err
	}

	err = parser.Parse(args)
	if err != nil {
		return config{}, err
	}

	if parser.Subcommand() == nil {
		parser.WriteHelp(os.Stdout)
		return config{}, fmt.Errorf("no valid subcommand provided")
	}

	cfg.authConfig = authConfig{
		excludeAzureCLIAuth:    cfg.ExcludeAzureCLIAuth,
		excludeEnvironmentAuth: cfg.ExcludeEnvironmentAuth,
		excludeMSIAuth:         cfg.ExcludeMSIAuth,
	}

	return cfg, err
}
