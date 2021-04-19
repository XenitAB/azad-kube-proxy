package actions

import (
	"context"
	"fmt"
	"net/url"
	"os/user"

	"github.com/go-logr/logr"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/customerrors"
)

type MenuConfig struct {
	discoverConfig *DiscoverConfig
	generateConfig *GenerateConfig
}

// NewMenuConfig ...
func NewMenuConfig(ctx context.Context, c *cli.Context) (MenuConfig, error) {
	discoverConfig, err := NewDiscoverConfig(ctx, c)
	if err != nil {
		return MenuConfig{}, err
	}

	generateConfig, err := NewGenerateConfig(ctx, c)
	if err != nil {
		return MenuConfig{}, err
	}

	return MenuConfig{
		&discoverConfig,
		&generateConfig,
	}, nil
}

// MenuFlags ...
func MenuFlags(ctx context.Context) []cli.Flag {
	usr, _ := user.Current()
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "kubeconfig",
			Usage:    "The path of the Kubernetes Config",
			EnvVars:  []string{"KUBECONFIG"},
			Value:    fmt.Sprintf("%s/.kube/config", usr.HomeDir),
			Required: false,
		},
		&cli.StringFlag{
			Name:     "tenant-id",
			Usage:    "Azure Tenant ID used with ENV auth",
			EnvVars:  []string{"AZURE_TENANT_ID"},
			Value:    "",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "client-id",
			Usage:    "Azure Client ID used with ENV auth",
			EnvVars:  []string{"AZURE_CLIENT_ID"},
			Value:    "",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "client-secret",
			Usage:    "Azure Client Secret used with ENV auth",
			EnvVars:  []string{"AZURE_CLIENT_SECRET"},
			Value:    "",
			Required: false,
		},
		&cli.BoolFlag{
			Name:    "exclude-azure-cli-auth",
			Usage:   "Should Azure CLI be excluded from the authentication?",
			EnvVars: []string{"EXCLUDE_AZURE_CLI_AUTH"},
			Value:   false,
		},
		&cli.BoolFlag{
			Name:    "exclude-environment-auth",
			Usage:   "Should environment be excluded from the authentication?",
			EnvVars: []string{"EXCLUDE_ENVIRONMENT_AUTH"},
			Value:   true,
		},
		&cli.BoolFlag{
			Name:    "exclude-msi-auth",
			Usage:   "Should MSI be excluded from the authentication?",
			EnvVars: []string{"EXCLUDE_MSI_AUTH"},
			Value:   true,
		},
	}
}

// Menu ...
func Menu(ctx context.Context, cfg MenuConfig) error {
	log := logr.FromContext(ctx)

	apps, err := runDiscover(ctx, *cfg.discoverConfig)
	if err != nil {
		log.V(1).Info("Unable to run discovery", "error", err.Error())
		return err
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "\U00002714 {{ .ClusterName | green }}",
		Inactive: "  {{ .ClusterName }}",
		Selected: "\U00002714 {{ .ClusterName | green }}",
		Details: `
--------- Cluster ----------
{{ "Cluster name:" | faint }}	{{ .ClusterName }}
{{ "Proxy URL:" | faint }}	{{ .ProxyURL }}
{{ "Resource URL:" | faint }}	{{ .Resource }}`,
	}

	clusterPrompt := promptui.Select{
		Label:     "Choose what cluster to configure",
		Items:     apps,
		Templates: templates,
	}

	idx, _, err := clusterPrompt.Run()

	if err != nil {
		log.V(1).Info("Unable to menu prompt", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeMenu, err)
	}

	cluster := apps[idx]
	proxyURL, err := url.Parse(cluster.ProxyURL)
	if err != nil {
		log.V(1).Info("Unable to parse Proxy URL", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeMenu, err)
	}

	cfg.generateConfig.Merge(GenerateConfig{
		clusterName: cluster.ClusterName,
		resource:    cluster.Resource,
		proxyURL:    *proxyURL,
		overwrite:   false,
	})

	generateCfg := *cfg.generateConfig

	err = Generate(ctx, generateCfg)

	if customerrors.To(err).ErrorType == customerrors.ErrorTypeOverwriteConfig {
		overwritePrompt := promptui.Select{
			Label: "Do you want to overwrite the config",
			Items: []string{"No", "Yes"},
		}

		_, result, err := overwritePrompt.Run()
		if err != nil {
			log.V(1).Info("Unable to menu prompt", "error", err.Error())
			return customerrors.New(customerrors.ErrorTypeMenu, err)
		}

		if result == "No" {
			err := fmt.Errorf("User selected not to overwrite config")
			log.V(1).Info("User selected not to overwrite config")
			return customerrors.New(customerrors.ErrorTypeOverwriteConfig, err)
		}

		cfg.generateConfig.Merge(GenerateConfig{
			clusterName: cluster.ClusterName,
			resource:    cluster.Resource,
			proxyURL:    *proxyURL,
			overwrite:   true,
		})

		generateCfg := *cfg.generateConfig

		err = Generate(ctx, generateCfg)
		if err != nil {
			log.V(1).Info("Unable to generate config", "error", err.Error())
			return err
		}

		return nil
	}

	if err != nil {
		log.V(1).Info("Unable to generate config", "error", err.Error())
		return err
	}

	return nil
}
