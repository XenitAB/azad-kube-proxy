package actions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/customerrors"
)

type MenuConfig struct {
	discoverConfig DiscoverConfig
}

// NewMenuConfig ...
func NewMenuConfig(ctx context.Context, c *cli.Context) (MenuConfig, error) {
	discoverConfig, err := NewDiscoverConfig(ctx, c)
	if err != nil {
		return MenuConfig{}, err
	}

	return MenuConfig{
		discoverConfig,
	}, nil
}

// MenuFlags ...
func MenuFlags(ctx context.Context) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "auth-method",
			Usage:    "Authentication method to use.",
			EnvVars:  []string{"AUTH_METHOD"},
			Value:    "CLI",
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

	// apps, err := runDiscover(ctx, cfg.discoverConfig)
	// if err != nil {
	// 	log.V(1).Info("Unable to run discovery", "error", err.Error())
	// 	return err
	// }

	appsTest := []discover{
		{
			ClusterName: "dev",
			ProxyURL:    "https://dev.example.com",
			Resource:    "https://dev.example.com",
		},
		{
			ClusterName: "qa",
			ProxyURL:    "https://qa.example.com",
			Resource:    "https://qa.example.com",
		},
		{
			ClusterName: "prod",
			ProxyURL:    "https://prod.example.com",
			Resource:    "https://prod.example.com",
		},
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "\U00002714 {{ .ClusterName | cyan }} ({{ .ProxyURL | red }})",
		Inactive: "  {{ .ClusterName | cyan }} ({{ .ProxyURL | red }})",
		Selected: "\U00002714 {{ .ClusterName | red | cyan }}",
		Details: `
--------- Cluster ----------
{{ "ClusterName:" | faint }}	{{ .ClusterName }}
{{ "ProxyURL:" | faint }}	{{ .ProxyURL }}
{{ "Resource:" | faint }}	{{ .Resource }}`,
	}

	prompt := promptui.Select{
		Label:     "Choose what cluster to configure",
		Items:     appsTest,
		Templates: templates,
	}

	_, result, err := prompt.Run()

	if err != nil {
		log.V(1).Info("Unable to menu prompt", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeMenu, err)
	}

	fmt.Printf("You choose %q\n", result)

	return nil
}
