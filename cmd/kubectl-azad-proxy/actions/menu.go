package actions

import (
	"context"
	"net/url"

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
	flags := mergeFlags(DiscoverFlags(ctx), GenerateFlags(ctx))
	flags = unrequireFlags(flags)

	return flags
}

// Menu ...
func Menu(ctx context.Context, cfg MenuConfig) error {
	log := logr.FromContext(ctx)

	// Run discovery of Azure AD applications
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

	// apps[idx] is the selected cluster
	cluster := apps[idx]
	proxyURL, err := url.Parse(cluster.ProxyURL)
	if err != nil {
		log.V(1).Info("Unable to parse Proxy URL", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeMenu, err)
	}

	// Update the generateConfig based on the selected cluster (overwrite = false)
	cfg.generateConfig.Merge(GenerateConfig{
		clusterName: cluster.ClusterName,
		resource:    cluster.Resource,
		proxyURL:    *proxyURL,
		overwrite:   false,
	})

	err = Generate(ctx, *cfg.generateConfig)

	// If the config already exists inside of KubeConfig, ask user if it should be overwritten
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

		// If user chose 'No' to overwrite, exit
		if result == "No" {
			log.V(0).Info("User selected not to overwrite config")
			return nil
		}

		// If user chose 'Yes', update config to allow overwrite and run again
		cfg.generateConfig.Merge(GenerateConfig{
			overwrite: true,
		})

		err = Generate(ctx, *cfg.generateConfig)
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

// unrequireFlags takes a []cli.Flag array 'f' and sets Required to false in all flags
func unrequireFlags(f []cli.Flag) []cli.Flag {
	flags := f
	for _, flag := range flags {
		switch flag := flag.(type) {
		case *cli.StringFlag:
			flag.Required = false
		case *cli.BoolFlag:
			flag.Required = false
		case *cli.IntFlag:
			flag.Required = false
		}
	}

	return flags
}

// mergeFlags taks two arrays ('a' and 'b') and removes any duplicates (based on the name) and outputs a merged array
func mergeFlags(a []cli.Flag, b []cli.Flag) []cli.Flag {
	flags := a

	for _, bFlag := range b {
		if !duplicateFlag(flags, bFlag) {
			flags = append(flags, bFlag)
		}
	}

	return flags
}

// duplicateFlag identified is the flag 'b' (based on the name) exists in the array 'a'
func duplicateFlag(a []cli.Flag, b cli.Flag) bool {
	duplicate := false

	for _, aFlag := range a {
		for _, aFlagName := range aFlag.Names() {
			for _, bFlagName := range b.Names() {
				if aFlagName == bFlagName {
					duplicate = true
					break
				}
			}
			if duplicate {
				break
			}
		}
		if duplicate {
			break
		}
	}

	return duplicate
}
