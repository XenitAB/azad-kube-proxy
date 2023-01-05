package main

import (
	"context"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/manifoldco/promptui"
)

type MenuClient struct {
	discoverClient DiscoverInterface
	generateClient GenerateInterface
	promptClient   promptInterface
}

func runMenu(ctx context.Context, cfg menuConfig, authCfg authConfig, promptClient promptInterface) error {
	client, err := newMenuClient(ctx, cfg, authCfg, promptClient)
	if err != nil {
		return err
	}

	return client.Menu(ctx)
}

func newMenuClient(ctx context.Context, cfg menuConfig, authCfg authConfig, promptClient promptInterface) (*MenuClient, error) {
	discoverCfg := discoverConfig{
		Output:            cfg.Output,
		AzureTenantID:     cfg.AzureTenantID,
		AzureClientID:     cfg.AzureClientID,
		AzureClientSecret: cfg.AzureClientSecret,
	}
	discoverClient, err := newDiscoverClient(ctx, discoverCfg, authCfg)
	if err != nil {
		return nil, err
	}

	generateCfg := generateConfig{
		ClusterName:           cfg.ClusterName,
		ProxyURL:              cfg.ProxyURL,
		Resource:              cfg.Resource,
		KubeConfig:            cfg.KubeConfig,
		TokenCacheDir:         cfg.TokenCacheDir,
		Overwrite:             cfg.Overwrite,
		TLSInsecureSkipVerify: cfg.TLSInsecureSkipVerify,
	}
	generateClient, err := newGenerateClient(ctx, generateCfg, authCfg)
	if err != nil {
		return nil, err
	}

	return &MenuClient{
		discoverClient,
		generateClient,
		promptClient,
	}, nil
}

// Menu ...
func (client *MenuClient) Menu(ctx context.Context) error {
	log := logr.FromContextOrDiscard(ctx)

	// Run discovery of Azure AD applications
	apps, err := client.discoverClient.Run(ctx)
	if err != nil {
		log.V(1).Info("Unable to run discovery", "error", err.Error())
		return err
	}

	cluster, err := client.promptClient.selectCluster(apps)
	if err != nil {
		log.V(1).Info("Unable to menu prompt", "error", err.Error())
		return newCustomError(errorTypeMenu, err)
	}

	proxyURL, err := url.Parse(cluster.ProxyURL)
	if err != nil {
		log.V(1).Info("Unable to parse Proxy URL", "error", err.Error())
		return newCustomError(errorTypeMenu, err)
	}

	// Update the GenerateClient based on the selected cluster (overwrite = false)
	client.generateClient.Merge(GenerateClient{
		clusterName: cluster.ClusterName,
		resource:    cluster.Resource,
		proxyURL:    *proxyURL,
		overwrite:   false,
	})

	err = client.generateClient.Generate(ctx)

	// If the config already exists inside of KubeConfig, ask user if it should be overwritten
	if toCustomError(err).errorType == errorTypeOverwriteConfig {
		overwrite, err := client.promptClient.overwriteConfig()
		if err != nil {
			log.V(1).Info("Unable to menu prompt", "error", err.Error())
			return newCustomError(errorTypeMenu, err)
		}

		// If user chose not to overwrite, exit
		if !overwrite {
			log.V(0).Info("User selected not to overwrite config")
			return nil
		}

		// If user chose 'Yes', update config to allow overwrite and run again
		client.generateClient.Merge(GenerateClient{
			overwrite: true,
		})

		err = client.generateClient.Generate(ctx)
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

type promptClient struct{}
type promptInterface interface {
	selectCluster(apps []discover) (discover, error)
	overwriteConfig() (bool, error)
}

func newPromptClient() promptInterface {
	return &promptClient{}
}

func (client *promptClient) selectCluster(apps []discover) (discover, error) {
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
		return discover{}, err
	}

	return apps[idx], nil
}

func (client *promptClient) overwriteConfig() (bool, error) {
	overwritePrompt := promptui.Select{
		Label: "Do you want to overwrite the config",
		Items: []string{"No", "Yes"},
	}

	_, result, err := overwritePrompt.Run()
	if err != nil {
		return false, err
	}

	// If user chose 'No' to overwrite, exit
	if result == "No" {
		return false, nil
	}

	return true, nil
}
