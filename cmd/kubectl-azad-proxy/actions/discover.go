package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	hamiltonAuth "github.com/manicminer/hamilton/auth"
	hamiltonClients "github.com/manicminer/hamilton/clients"
	hamiltonEnvironments "github.com/manicminer/hamilton/environments"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

var (
	azureADAppTag = "azad-kube-proxy"
)

type outputType string

var tableOutputType outputType = "TABLE"
var jsonOutputType outputType = "JSON"

type discover struct {
	ClusterName string `json:"cluster_name"`
	Resource    string `json:"resource"`
}

// DiscoverConfig ...
type DiscoverConfig struct {
	outputType
	tenantID               string
	clientID               string
	clientSecret           string
	enableClientSecretAuth bool
	enableAzureCliToken    bool
	enableMsiAuth          bool
}

// NewDiscoverConfig ...
func NewDiscoverConfig(ctx context.Context, c *cli.Context) (DiscoverConfig, error) {
	var output outputType
	switch c.String("output") {
	case "TABLE":
		output = tableOutputType
	case "JSON":
		output = jsonOutputType
	default:
		return DiscoverConfig{}, fmt.Errorf("Supported outputs are TABLE and JSON. The following was used: %s", c.String("output"))
	}

	enableAzureCliToken := !c.Bool("exclude-azure-cli-auth")
	tenantID := c.String("tenant-id")
	if tenantID == "" && enableAzureCliToken {
		cliConfig, err := hamiltonAuth.NewAzureCliConfig(hamiltonAuth.MsGraph, "")
		if err != nil {
			return DiscoverConfig{}, err
		}

		tenantID = cliConfig.TenantID
		if tenantID == "" {
			return DiscoverConfig{}, fmt.Errorf("No tenantID could be extracted from Azure CLI authentication")
		}
	}

	return DiscoverConfig{
		outputType:             output,
		tenantID:               tenantID,
		clientID:               c.String("client-id"),
		clientSecret:           c.String("client-secret"),
		enableClientSecretAuth: !c.Bool("exclude-environment-auth"),
		enableAzureCliToken:    enableAzureCliToken,
		enableMsiAuth:          !c.Bool("exclude-msi-auth"),
	}, nil
}

// DiscoverFlags ...
func DiscoverFlags(ctx context.Context) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "output",
			Usage:    "How to output the data",
			EnvVars:  []string{"OUTPUT"},
			Value:    "TABLE",
			Required: false,
		},
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
			Value:   false,
		},
		&cli.BoolFlag{
			Name:    "exclude-msi-auth",
			Usage:   "Should MSI be excluded from the authentication?",
			EnvVars: []string{"EXCLUDE_MSI_AUTH"},
			Value:   false,
		},
	}
}

// Discover ...
func Discover(ctx context.Context, cfg DiscoverConfig) (string, error) {
	authConfig := &hamiltonAuth.Config{
		Environment:            hamiltonEnvironments.Global,
		TenantID:               cfg.tenantID,
		ClientID:               cfg.clientID,
		ClientSecret:           cfg.clientSecret,
		EnableClientSecretAuth: cfg.enableClientSecretAuth,
		EnableAzureCliToken:    cfg.enableAzureCliToken,
		EnableMsiAuth:          cfg.enableMsiAuth,
	}

	authorizer, err := authConfig.NewAuthorizer(ctx, hamiltonAuth.MsGraph)
	if err != nil {
		return "", err
	}
	appsClient := hamiltonClients.NewApplicationsClient(cfg.tenantID)
	appsClient.BaseClient.Authorizer = authorizer

	graphFilter := fmt.Sprintf("tags/any(s: s eq '%s')", azureADAppTag)

	clusterApps, _, err := appsClient.List(ctx, graphFilter)
	if err != nil {
		return "", err
	}

	discoverData := []discover{}

	for _, clusterApp := range *clusterApps {
		if len(*clusterApp.IdentifierUris) != 0 {
			displayName := *clusterApp.DisplayName
			appUris := *clusterApp.IdentifierUris
			discoverData = append(discoverData, discover{
				ClusterName: displayName,
				Resource:    appUris[0],
			})
		}
	}

	var output string
	switch cfg.outputType {
	case tableOutputType:
		output = getTable(discoverData)
	case jsonOutputType:
		output, err = getJSON(discoverData)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("Unknown output type: %s", cfg.outputType)
	}

	return output, nil
}

func getTable(discoverData []discover) string {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Cluster Name", "Resource"})

	data := [][]string{}
	for _, d := range discoverData {
		data = append(data, []string{
			d.ClusterName,
			d.Resource,
		})
	}

	for _, v := range data {
		table.Append(v)
	}

	table.Render()

	return tableString.String()
}

func getJSON(discoverData []discover) (string, error) {
	output, err := json.MarshalIndent(discoverData, "", "  ")
	if err != nil {
		return "", err
	}
	return string(output), nil
}
