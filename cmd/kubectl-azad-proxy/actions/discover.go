package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	hamiltonAuth "github.com/manicminer/hamilton/auth"
	hamiltonEnvironments "github.com/manicminer/hamilton/environments"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"github.com/xenitab/azad-kube-proxy/cmd/kubectl-azad-proxy/customerrors"
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
	ProxyURL    string `json:"proxy_url"`
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
	log := logr.FromContext(ctx)

	var output outputType
	switch strings.ToUpper(c.String("output")) {
	case "TABLE":
		output = tableOutputType
	case "JSON":
		output = jsonOutputType
	default:
		err := fmt.Errorf("Supported outputs are TABLE and JSON. The following was used: %s", c.String("output"))
		log.V(1).Info("Unsupported output", "error", err.Error())
		return DiscoverConfig{}, err
	}

	enableAzureCliToken := !c.Bool("exclude-azure-cli-auth")
	tenantID := c.String("tenant-id")
	if tenantID == "" && enableAzureCliToken {
		cliConfig, err := hamiltonAuth.NewAzureCliConfig(hamiltonAuth.MsGraph, "")
		if err != nil {
			log.V(1).Info("Unable to create CliConfig", "error", err.Error())
			return DiscoverConfig{}, customerrors.New(customerrors.ErrorTypeAuthentication, err)
		}

		tenantID = cliConfig.TenantID
		if tenantID == "" {
			err := fmt.Errorf("No tenantID could be extracted from Azure CLI authentication")
			log.V(1).Info("No tenantID", "error", err.Error())
			return DiscoverConfig{}, customerrors.New(customerrors.ErrorTypeAuthentication, err)
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

// Discover ...
func Discover(ctx context.Context, cfg DiscoverConfig) (string, error) {
	log := logr.FromContext(ctx)

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
		log.V(1).Info("Unable to create authorizer", "error", err.Error())
		return "", customerrors.New(customerrors.ErrorTypeAuthentication, err)
	}
	appsClient := hamiltonMsgraph.NewApplicationsClient(cfg.tenantID)
	appsClient.BaseClient.Authorizer = authorizer

	graphFilter := fmt.Sprintf("tags/any(s: s eq '%s')", azureADAppTag)

	clusterApps, resCode, err := appsClient.List(ctx, graphFilter)
	if err != nil {
		log.V(1).Info("Unable to to list Azure AD applications", "error", err.Error(), "responseCode", resCode)
		return "", customerrors.New(customerrors.ErrorTypeAuthorization, err)
	}

	discoverData := getDiscoverData(*clusterApps)

	var output string
	switch cfg.outputType {
	case tableOutputType:
		output = getTable(discoverData)
	case jsonOutputType:
		output, err = getJSON(discoverData)
		if err != nil {
			log.V(1).Info("Unable to convert output to JSON", "error", err.Error())
			return "", err
		}
	default:
		return "", fmt.Errorf("Unknown output type: %s", cfg.outputType)
	}

	return output, nil
}

func getDiscoverData(clusterApps []hamiltonMsgraph.Application) []discover {
	discoverData := []discover{}

	for _, clusterApp := range clusterApps {
		var clusterName, resource, proxyURL string
		if len(*clusterApp.IdentifierUris) != 0 {
			for _, tag := range *clusterApp.Tags {
				if strings.HasPrefix(tag, "cluster_name:") {
					tagClusterName := strings.Replace(tag, "cluster_name:", "", 1)
					if len(tagClusterName) != 0 {
						clusterName = tagClusterName
					}
				}

				if strings.HasPrefix(tag, "proxy_url:") {
					tagProxyURL := strings.Replace(tag, "proxy_url:", "", 1)
					if len(tagProxyURL) != 0 {
						proxyURL = tagProxyURL
					}
				}
			}

			appUris := *clusterApp.IdentifierUris
			resource = appUris[0]

			if clusterName == "" {
				clusterName = *clusterApp.DisplayName
			}

			if proxyURL == "" {
				proxyURL = resource
			}

			discoverData = append(discoverData, discover{
				ClusterName: clusterName,
				Resource:    resource,
				ProxyURL:    proxyURL,
			})
		}
	}

	return discoverData
}

func getTable(discoverData []discover) string {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"Cluster Name", "Resource", "Proxy URL"})

	data := [][]string{}
	for _, d := range discoverData {
		data = append(data, []string{
			d.ClusterName,
			d.Resource,
			d.ProxyURL,
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
