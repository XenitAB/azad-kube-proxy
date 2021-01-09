package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/to"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/go-logr/logr"
	"github.com/jongio/azidext/go/azidext"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// Client ...
type Client struct {
	ClientID        string
	ClientSecret    string
	TenantID        string
	GraphFilter     string
	Cache           cache.Cache
	AzureCredential *azidentity.ClientSecretCredential
	UsersClient     graphrbac.UsersClient
	GroupsClient    graphrbac.GroupsClient
	msGraphToken    *azcore.AccessToken
}

// NewAzureClient returns an Azure client or error
func NewAzureClient(ctx context.Context, clientID, clientSecret, tenantID, graphFilter string, c cache.Cache) (Client, error) {
	a := Client{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
		Cache:        c,
	}

	var err error

	a.AzureCredential, err = a.getAzureCredential(ctx)
	if err != nil {
		return Client{}, err
	}

	a.UsersClient, err = a.getAzureADUsersClient(ctx)
	if err != nil {
		return Client{}, err

	}

	a.GroupsClient, err = a.getAzureADGroupsClient(ctx)
	if err != nil {
		return Client{}, err
	}

	a.msGraphToken, err = a.GetMSGraphToken(ctx)
	if err != nil {
		return Client{}, err
	}

	if graphFilter != "" {
		graphFilter = fmt.Sprintf("startswith(displayName,'%s')", graphFilter)
	}
	a.GraphFilter = graphFilter

	return a, nil
}

// GetUserGroupsFromCache returns the group names the user is a member of
func (client *Client) GetUserGroupsFromCache(ctx context.Context, objectID string, userType models.UserType) ([]models.Group, error) {
	var groupIDs []string
	var err error

	switch userType {
	case models.NormalUserType:
		groupIDs, err = client.getUserGroups(ctx, objectID)
		if err != nil {
			return nil, err
		}
	case models.ServicePrincipalUserType:
		groupIDs, err = client.getServicePrincipalGroups(ctx, objectID)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("Unknown userType: %s", userType)
	}

	var groupNames []models.Group
	for _, groupID := range groupIDs {
		group, found, err := client.Cache.GetGroup(ctx, groupID)
		if err != nil {
			return nil, err
		}
		if found {
			groupNames = append(groupNames, group)
		}
	}

	return groupNames, nil
}

func (client *Client) getUserGroups(ctx context.Context, objectID string) ([]string, error) {
	log := logr.FromContext(ctx)

	groupsResponse, err := client.UsersClient.GetMemberGroups(ctx, objectID, graphrbac.UserGetMemberGroupsParameters{
		SecurityEnabledOnly: to.BoolPtr(false),
	})
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for user", "objectID", objectID)
		return nil, err
	}

	return *groupsResponse.Value, nil
}

func (client *Client) getServicePrincipalGroups(ctx context.Context, objectID string) ([]string, error) {
	log := logr.FromContext(ctx)

	url, err := url.Parse(fmt.Sprintf("https://graph.microsoft.com/v1.0/servicePrincipals/%s/transitiveMemberOf/microsoft.graph.group?$select=id", objectID))
	if err != nil {
		log.Error(err, "Unable to parse URL")
		return nil, err
	}

	msGraphToken, err := client.GetMSGraphToken(ctx)
	if err != nil {
		log.Error(err, "Unable to get MS Graph Token")
		return nil, err
	}

	req := &http.Request{
		Method: "GET",
		URL:    url,
		Header: map[string][]string{
			"Authorization": {fmt.Sprintf("Bearer %s", msGraphToken.Token)},
			"Content-type":  {"application/json"},
		},
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for service principal", "objectID", objectID)
		return nil, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err, "Unable to read body of response", "objectID", objectID)
		return nil, err
	}

	type responseValue struct {
		ID string `json:"id"`
	}

	var responseData struct {
		Values []responseValue `json:"value"`
	}

	json.Unmarshal(body, &responseData)

	var groups []string
	for _, group := range responseData.Values {
		groups = append(groups, group.ID)
	}

	return groups, nil
}

func (client *Client) getAllGroups(ctx context.Context) ([]graphrbac.ADGroup, error) {
	log := logr.FromContext(ctx)

	var groups []graphrbac.ADGroup
	for list, err := client.GroupsClient.List(ctx, client.GraphFilter); list.NotDone(); err = list.NextWithContext(ctx) {
		if err != nil {
			log.Error(err, "Unable to list Azure AD groups", "graphFilter", client.GraphFilter)
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

	cred, err := azidentity.NewClientSecretCredential(client.TenantID, client.ClientID, client.ClientSecret, nil)
	if err != nil {
		log.Error(err, "azidentity.NewClientSecretCredential")
		return nil, err
	}
	return cred, nil
}

// GetMSGraphToken generates a new token if none exists and an existing if it not expired
func (client *Client) GetMSGraphToken(ctx context.Context) (*azcore.AccessToken, error) {
	log := logr.FromContext(ctx)

	generateNewToken := true
	if client.msGraphToken != nil {
		if client.msGraphToken.ExpiresOn.After(time.Now().Add(-5 * time.Minute)) {
			generateNewToken = false
		}
	}

	if generateNewToken {
		token, err := client.AzureCredential.GetToken(ctx, azcore.TokenRequestOptions{Scopes: []string{"https://graph.microsoft.com/.default"}})
		if err != nil {
			log.Error(err, "client.AzureCredential.GetToken")
			return nil, err
		}

		return token, nil
	}

	return client.msGraphToken, nil
}

func (client *Client) getGraphAuthorizer(ctx context.Context) (autorest.Authorizer, error) {
	log := logr.FromContext(ctx)

	cred, err := azidentity.NewClientSecretCredential(client.TenantID, client.ClientID, client.ClientSecret, nil)
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
	groupsClient := graphrbac.NewGroupsClient(client.TenantID)
	authorizer, err := client.getGraphAuthorizer(ctx)
	if err != nil {
		return graphrbac.GroupsClient{}, err
	}

	groupsClient.Authorizer = authorizer

	return groupsClient, nil
}

func (client *Client) getAzureADUsersClient(ctx context.Context) (graphrbac.UsersClient, error) {
	usersClient := graphrbac.NewUsersClient(client.TenantID)
	authorizer, err := client.getGraphAuthorizer(ctx)
	if err != nil {
		return graphrbac.UsersClient{}, err
	}

	usersClient.Authorizer = authorizer

	return usersClient, nil
}

// StartSyncTickerAzureADGroups initiates a ticker that will sync Azure AD Groups
func (client *Client) StartSyncTickerAzureADGroups(ctx context.Context, syncInterval time.Duration) (*time.Ticker, chan bool, error) {
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

func (client *Client) syncAzureADGroupsCache(ctx context.Context, syncReason string) error {
	log := logr.FromContext(ctx)

	groups, err := client.getAllGroups(ctx)
	if err != nil {
		log.Error(err, "Unable to syncronize groups")
		return err
	}

	for _, group := range groups {
		_, found, err := client.Cache.GetGroup(ctx, *group.ObjectID)
		if err != nil {
			return err
		}
		if !found {
			client.Cache.SetGroup(ctx, *group.ObjectID, models.Group{Name: *group.DisplayName})
		}
	}

	log.Info("Synchronized Azure AD groups to cache", "groupCount", len(groups), "syncReason", syncReason)

	return nil
}
