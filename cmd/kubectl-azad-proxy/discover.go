package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/go-logr/logr"
	hamiltonAuth "github.com/manicminer/hamilton/auth"
	hamiltonEnvironments "github.com/manicminer/hamilton/environments"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
	hamiltonOdata "github.com/manicminer/hamilton/odata"
	"github.com/olekukonko/tablewriter"
)

const (
	azureADAppTag         = "azad-kube-proxy"
	tagRestApiVersion     = "2021-04-01"
	tagSubscriptionPrefix = "_azad-kube-proxy"
)

type outputType string

var tableOutputType outputType = "TABLE"
var jsonOutputType outputType = "JSON"

type discover struct {
	ClusterName string `json:"cluster_name"`
	Resource    string `json:"resource"`
	ProxyURL    string `json:"proxy_url"`
}

type DiscoverClient struct {
	outputType
	tenantID               string
	clientID               string
	clientSecret           string
	enableClientSecretAuth bool
	enableAzureCliToken    bool
	enableMsiAuth          bool
}

type DiscoverInterface interface {
	Discover(ctx context.Context) (string, error)
	Run(ctx context.Context) ([]discover, error)
}

func runDiscover(ctx context.Context, writer io.Writer, cfg discoverConfig, authCfg authConfig) error {
	client, err := newDiscoverClient(ctx, cfg, authCfg)
	if err != nil {
		return err
	}

	output, err := client.Run(ctx)
	if err != nil {
		return err
	}

	fmt.Fprint(writer, output)

	return nil
}

func newDiscoverClient(ctx context.Context, cfg discoverConfig, authCfg authConfig) (*DiscoverClient, error) {
	log := logr.FromContextOrDiscard(ctx)

	var output outputType
	switch strings.ToUpper(cfg.Output) {
	case "TABLE":
		output = tableOutputType
	case "JSON":
		output = jsonOutputType
	default:
		err := fmt.Errorf("Supported outputs are TABLE and JSON. The following was used: %s", cfg.Output)
		log.V(1).Info("Unsupported output", "error", err.Error())
		return nil, err
	}

	enableAzureCliToken := !authCfg.excludeAzureCLIAuth
	tenantID := cfg.AzureTenantID
	if tenantID == "" && enableAzureCliToken {
		cliConfig, err := hamiltonAuth.NewAzureCliConfig(hamiltonEnvironments.MsGraphGlobal, "")
		if err != nil {
			log.V(1).Info("Unable to create CliConfig", "error", err.Error())
			return nil, newCustomError(errorTypeAuthentication, err)
		}

		tenantID = cliConfig.TenantID
		if tenantID == "" {
			err := fmt.Errorf("No tenantID could be extracted from Azure CLI authentication")
			log.V(1).Info("No tenantID", "error", err.Error())
			return nil, newCustomError(errorTypeAuthentication, err)
		}
	}

	return &DiscoverClient{
		outputType:             output,
		tenantID:               tenantID,
		clientID:               cfg.AzureClientID,
		clientSecret:           cfg.AzureClientSecret,
		enableClientSecretAuth: !authCfg.excludeEnvironmentAuth,
		enableAzureCliToken:    enableAzureCliToken,
		enableMsiAuth:          !authCfg.excludeMSIAuth,
	}, nil
}

func (client *DiscoverClient) Discover(ctx context.Context) (string, error) {
	log := logr.FromContextOrDiscard(ctx)

	apps, err := client.Run(ctx)
	if err != nil {
		return "", err
	}

	var output string
	switch client.outputType {
	case tableOutputType:
		output = getTable(apps)
	case jsonOutputType:
		output, err = getJSON(apps)
		if err != nil {
			log.V(1).Info("Unable to convert output to JSON", "error", err.Error())
			return "", err
		}
	default:
		return "", fmt.Errorf("Unknown output type: %s", client.outputType)
	}

	return output, nil
}

func (client *DiscoverClient) Run(ctx context.Context) ([]discover, error) {
	log := logr.FromContextOrDiscard(ctx)

	authConfig := &hamiltonAuth.Config{
		Environment:            hamiltonEnvironments.Global,
		TenantID:               client.tenantID,
		ClientID:               client.clientID,
		ClientSecret:           client.clientSecret,
		EnableClientSecretAuth: client.enableClientSecretAuth,
		EnableAzureCliToken:    client.enableAzureCliToken,
		EnableMsiAuth:          client.enableMsiAuth,
	}

	authorizer, err := authConfig.NewAuthorizer(ctx, hamiltonEnvironments.MsGraphGlobal)
	if err != nil {
		log.V(1).Info("Unable to create authorizer", "error", err.Error())
		return []discover{}, newCustomError(errorTypeAuthentication, err)
	}

	appsClient := hamiltonMsgraph.NewApplicationsClient(client.tenantID)
	appsClient.BaseClient.Authorizer = authorizer

	graphFilter := fmt.Sprintf("tags/any(s: s eq '%s')", azureADAppTag)

	odataQuery := hamiltonOdata.Query{
		Filter: graphFilter,
	}

	clusterApps, resCode, err := appsClient.List(ctx, odataQuery)
	if err != nil {
		if strings.Contains(err.Error(), "Insufficient privileges to complete the operation") {
			return client.trySubscriptionsDiscovery(ctx)
		}
		log.V(1).Info("Unable to to list Azure AD applications", "error", err.Error(), "responseCode", resCode)
		return []discover{}, newCustomError(errorTypeAuthorization, err)
	}

	discoverData := getDiscoverData(*clusterApps)

	return discoverData, nil
}

func (client *DiscoverClient) trySubscriptionsDiscovery(ctx context.Context) ([]discover, error) {
	log := logr.FromContextOrDiscard(ctx)

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, newCustomError(errorTypeAuthorization, err)
	}

	subscriptionClient, err := armsubscriptions.NewClient(cred, nil)
	if err != nil {
		return nil, newCustomError(errorTypeAuthorization, err)
	}

	subscriptionsIds := []string{}
	pager := subscriptionClient.NewListPager(nil)
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, newCustomError(errorTypeAuthorization, err)
		}
		for _, v := range nextResult.Value {
			if v.SubscriptionID == nil {
				continue
			}
			subscriptionsIds = append(subscriptionsIds, *v.SubscriptionID)
		}
	}

	for _, subscriptionId := range subscriptionsIds {
		log.V(1).Info("Trying to find clusters on subscription", "subscription_id", subscriptionId)
		clusters, err := client.trySubscriptionDiscovery(ctx, cred, subscriptionId)
		if err == nil {
			return clusters, nil
		}
	}

	return nil, newCustomError(errorTypeAuthentication, fmt.Errorf("unable to find any clusters on any subscriptions"))
}

func (client *DiscoverClient) trySubscriptionDiscovery(ctx context.Context, cred *azidentity.DefaultAzureCredential, subscriptionId string) ([]discover, error) {
	tagClient, err := armresources.NewClient(subscriptionId, cred, nil)
	if err != nil {
		return nil, err
	}
	res, err := tagClient.GetByID(ctx, fmt.Sprintf("/subscriptions/%s", subscriptionId), tagRestApiVersion, &armresources.ClientGetByIDOptions{})
	if err != nil {
		return nil, err
	}

	clusters := []discover{}
	for key, value := range res.Tags {
		if value == nil {
			continue
		}
		if !strings.HasPrefix(key, tagSubscriptionPrefix) {
			continue
		}
		cluster := discover{}
		err := json.Unmarshal([]byte(*value), &cluster)
		if err != nil {
			return nil, err
		}

		if cluster.ClusterName == "" || cluster.ProxyURL == "" || cluster.Resource == "" {
			return nil, fmt.Errorf("all fields of the cluster not found: %v", cluster)
		}
		clusters = append(clusters, cluster)
	}

	if len(clusters) == 0 {
		return nil, fmt.Errorf("no clusters found on subscription")
	}

	return clusters, nil
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
