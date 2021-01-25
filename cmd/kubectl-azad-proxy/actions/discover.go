package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	hamiltonAuth "github.com/manicminer/hamilton/auth"
	hamiltonClients "github.com/manicminer/hamilton/clients"
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
	outputType outputType
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

	return DiscoverConfig{
		outputType: output,
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
	}
}

// Discover ...
func Discover(ctx context.Context, cfg DiscoverConfig) (string, error) {
	azureCliConfig, err := hamiltonAuth.NewAzureCliConfig(hamiltonAuth.MsGraph, "")
	if err != nil {
		return "", err
	}

	authorizer, err := hamiltonAuth.NewAzureCliAuthorizer(ctx, hamiltonAuth.MsGraph, "")
	if err != nil {
		return "", err
	}

	appsClient := hamiltonClients.NewApplicationsClient(azureCliConfig.TenantID)
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
