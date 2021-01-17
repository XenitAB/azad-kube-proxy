package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/go-logr/logr"
	"github.com/jongio/azidext/go/azidext"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type userInterface interface {
	getGroups(ctx context.Context, objectID string) ([]models.Group, error)
}

// ClientInterface ...
type ClientInterface interface {
	GetUserGroups(ctx context.Context, objectID string, userType models.UserType) ([]models.Group, error)
	StartSyncGroups(ctx context.Context, syncInterval time.Duration) (*time.Ticker, chan bool, error)
}

// Client ...
type Client struct {
	clientID             string
	clientSecret         string
	tenantID             string
	graphFilter          string
	cacheClient          cache.ClientInterface
	groupsClient         graphrbac.GroupsClient
	user                 *user
	servicePrincipalUser *servicePrincipalUser
}

// NewAzureClient returns an Azure client or error
func NewAzureClient(ctx context.Context, clientID, clientSecret, tenantID, graphFilter string, cacheClient cache.ClientInterface) (*Client, error) {
	a := &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		tenantID:     tenantID,
		cacheClient:  cacheClient,
	}

	var err error

	azureCredential, err := a.getAzureCredential(ctx)
	if err != nil {
		return nil, err
	}

	usersClient, err := a.getAzureADUsersClient(ctx)
	if err != nil {
		return nil, err
	}

	a.groupsClient, err = a.getAzureADGroupsClient(ctx)
	if err != nil {
		return nil, err
	}

	if graphFilter != "" {
		graphFilter = fmt.Sprintf("startswith(displayName,'%s')", graphFilter)
	}
	a.graphFilter = graphFilter

	a.user, err = newUser(ctx, cacheClient, usersClient)
	if err != nil {
		return nil, err
	}

	a.servicePrincipalUser, err = newServicePrincipalUser(ctx, azureCredential, cacheClient)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// GetUserGroups ...
func (client *Client) GetUserGroups(ctx context.Context, objectID string, userType models.UserType) ([]models.Group, error) {
	var user userInterface

	switch userType {
	case models.NormalUserType:
		user = client.user
	case models.ServicePrincipalUserType:
		user = client.servicePrincipalUser
	default:
		return nil, fmt.Errorf("Unknown userType: %s", userType)
	}

	return user.getGroups(ctx, objectID)
}

// StartSyncGroups initiates a ticker that will sync Azure AD Groups
func (client *Client) StartSyncGroups(ctx context.Context, syncInterval time.Duration) (*time.Ticker, chan bool, error) {
	log := logr.FromContext(ctx)

	ticker := time.NewTicker(syncInterval)
	syncChan := make(chan bool)

	err := client.syncAzureADGroupsCache(ctx, "initial")
	if err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			select {
			case <-syncChan:
				log.Info("Stopped StartSyncTickerAzureADGroups")
				return
			case _ = <-ticker.C:
				_ = client.syncAzureADGroupsCache(ctx, "ticker")
			}
		}
	}()

	return ticker, syncChan, nil
}

func (client *Client) getAllGroups(ctx context.Context) ([]graphrbac.ADGroup, error) {
	log := logr.FromContext(ctx)

	var groups []graphrbac.ADGroup
	for list, err := client.groupsClient.List(ctx, client.graphFilter); list.NotDone(); err = list.NextWithContext(ctx) {
		if err != nil {
			log.Error(err, "Unable to list Azure AD groups", "graphFilter", client.graphFilter)
			return nil, err
		}
		for _, group := range list.Values() {
			groups = append(groups, group)
		}
	}

	return groups, nil
}

func (client *Client) getAzureCredential(ctx context.Context) (*azidentity.ClientSecretCredential, error) {
	log := logr.FromContext(ctx)

	cred, err := azidentity.NewClientSecretCredential(client.tenantID, client.clientID, client.clientSecret, nil)
	if err != nil {
		log.Error(err, "azidentity.NewClientSecretCredential")
		return nil, err
	}
	return cred, nil
}

func (client *Client) getGraphAuthorizer(ctx context.Context) (autorest.Authorizer, error) {
	log := logr.FromContext(ctx)

	cred, err := azidentity.NewClientSecretCredential(client.tenantID, client.clientID, client.clientSecret, nil)
	if err != nil {
		log.Error(err, "azidentity.NewClientSecretCredential")
		return nil, err
	}

	authorizer := azidext.NewAzureIdentityCredentialAdapter(
		cred,
		azcore.AuthenticationPolicyOptions{
			Options: azcore.TokenRequestOptions{
				Scopes: []string{"https://graph.windows.net/.default"}}})

	return authorizer, nil
}

func (client *Client) getAzureADGroupsClient(ctx context.Context) (graphrbac.GroupsClient, error) {
	groupsClient := graphrbac.NewGroupsClient(client.tenantID)
	authorizer, err := client.getGraphAuthorizer(ctx)
	if err != nil {
		return graphrbac.GroupsClient{}, err
	}

	groupsClient.Authorizer = authorizer

	return groupsClient, nil
}

func (client *Client) getAzureADUsersClient(ctx context.Context) (graphrbac.UsersClient, error) {
	usersClient := graphrbac.NewUsersClient(client.tenantID)
	authorizer, err := client.getGraphAuthorizer(ctx)
	if err != nil {
		return graphrbac.UsersClient{}, err
	}

	usersClient.Authorizer = authorizer

	return usersClient, nil
}

func (client *Client) syncAzureADGroupsCache(ctx context.Context, syncReason string) error {
	log := logr.FromContext(ctx)

	groups, err := client.getAllGroups(ctx)
	if err != nil {
		log.Error(err, "Unable to syncronize groups")
		return err
	}

	for _, group := range groups {
		_, found, err := client.cacheClient.GetGroup(ctx, *group.ObjectID)
		if err != nil {
			return err
		}
		if !found {
			client.cacheClient.SetGroup(ctx, *group.ObjectID, models.Group{Name: *group.DisplayName})
		}
	}

	log.Info("Synchronized Azure AD groups to cache", "groupCount", len(groups), "syncReason", syncReason)

	return nil
}
