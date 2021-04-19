package actions

import (
	"context"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/customerrors"
)

type MenuClient struct {
	discoverClient DiscoverInterface
	generateClient GenerateInterface
	promptClient   promptInterface
}

type MenuInterface interface {
	Menu(ctx context.Context) error
}

// NewMenuClient ...
func NewMenuClient(ctx context.Context, c *cli.Context) (MenuInterface, error) {
	discoverClient, err := NewDiscoverClient(ctx, c)
	if err != nil {
		return nil, err
	}

	generateClient, err := NewGenerateClient(ctx, c)
	if err != nil {
		return nil, err
	}

	promptClient := newPromptClient()

	return &MenuClient{
		discoverClient,
		generateClient,
		promptClient,
	}, nil
}

// MenuFlags ...
func MenuFlags(ctx context.Context) []cli.Flag {
	flags := mergeFlags(DiscoverFlags(ctx), GenerateFlags(ctx))
	flags = unrequireFlags(flags)

	return flags
}

// Menu ...
func (client *MenuClient) Menu(ctx context.Context) error {
	log := logr.FromContext(ctx)

	// Run discovery of Azure AD applications
	apps, err := client.discoverClient.Run(ctx)
	if err != nil {
		log.V(1).Info("Unable to run discovery", "error", err.Error())
		return err
	}

	cluster, err := client.promptClient.selectCluster(apps)
	if err != nil {
		log.V(1).Info("Unable to menu prompt", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeMenu, err)
	}

	proxyURL, err := url.Parse(cluster.ProxyURL)
	if err != nil {
		log.V(1).Info("Unable to parse Proxy URL", "error", err.Error())
		return customerrors.New(customerrors.ErrorTypeMenu, err)
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
	if customerrors.To(err).ErrorType == customerrors.ErrorTypeOverwriteConfig {
		overwrite, err := client.promptClient.overwriteConfig()
		if err != nil {
			log.V(1).Info("Unable to menu prompt", "error", err.Error())
			return customerrors.New(customerrors.ErrorTypeMenu, err)
		}

		// If user chose 'No' to overwrite, exit
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

// unrequireFlags takes a []cli.Flag array 'f' and sets Required to false in all flags
func unrequireFlags(f []cli.Flag) []cli.Flag {
	flags := f
	for _, flag := range flags {
		switch flag := flag.(type) {
		case *cli.StringFlag:
			flag.Required = false
		case *cli.BoolFlag:
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

	return false, nil
}
