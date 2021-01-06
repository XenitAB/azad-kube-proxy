package azure

import (
	"context"
	"fmt"
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

// Client is the Azure Client interface
type Client interface {
	GetUserGroupsFromCache(userObjectID string) ([]models.Group, error)
	StartSyncTickerAzureADGroups(syncInterval time.Duration) (*time.Ticker, chan bool, error)
}

// Azure ...
type Azure struct {
	Context      context.Context
	ClientID     string
	ClientSecret string
	TenantID     string
	GraphFilter  string
	Cache        cache.Client
	UsersClient  graphrbac.UsersClient
	GroupsClient graphrbac.GroupsClient
}

// NewAzureClient returns an Azure client or error
func NewAzureClient(ctx context.Context, clientID, clientSecret, tenantID, graphFilter string, c cache.Client) (Azure, error) {
	a := Azure{
		Context:      ctx,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
		Cache:        c,
	}

	var err error

	a.UsersClient, err = a.getAzureADUsersClient()
	if err != nil {
		return Azure{}, err

	}

	a.GroupsClient, err = a.getAzureADGroupsClient()
	if err != nil {
		return Azure{}, err
	}

	if graphFilter != "" {
		graphFilter = fmt.Sprintf("startswith(displayName,'%s')", graphFilter)
	}
	a.GraphFilter = graphFilter

	return a, nil

}

// GetUserGroupsFromCache returns the group names the user is a member of
func (a *Azure) GetUserGroupsFromCache(userObjectID string, username string, tokenGroups []string) ([]models.Group, error) {
	var groupIDs []string
	var err error
	if username != "" {
		groupIDs, err = a.getUserGroups(userObjectID)
		if err != nil {
			return nil, err
		}
	}

	if username == "" {
		groupIDs = tokenGroups
	}

	var groupNames []models.Group
	for _, groupID := range groupIDs {
		group, found, err := a.Cache.GetGroup(groupID)
		if err != nil {
			return nil, err
		}
		if found {
			groupNames = append(groupNames, group)
		}
	}

	return groupNames, nil
}

func (a *Azure) getUserGroups(userObjectID string) ([]string, error) {
	log := logr.FromContext(a.Context)

	groupsResponse, err := a.UsersClient.GetMemberGroups(a.Context, userObjectID, graphrbac.UserGetMemberGroupsParameters{
		SecurityEnabledOnly: to.BoolPtr(false),
	})
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for user", "userObjectID", userObjectID)
		return nil, err
	}

	return *groupsResponse.Value, nil
}

func (a *Azure) getAllGroups() ([]graphrbac.ADGroup, error) {
	log := logr.FromContext(a.Context)

	var groups []graphrbac.ADGroup
	for list, err := a.GroupsClient.List(a.Context, a.GraphFilter); list.NotDone(); err = list.NextWithContext(a.Context) {
		if err != nil {
			log.Error(err, "Unable to list Azure AD groups", "graphFilter", a.GraphFilter)
			return nil, err
		}
		for _, group := range list.Values() {
			groups = append(groups, group)
		}
	}

	return groups, nil
}

func (a *Azure) getGraphAuthorizer() (autorest.Authorizer, error) {
	log := logr.FromContext(a.Context)

	cred, err := azidentity.NewClientSecretCredential(a.TenantID, a.ClientID, a.ClientSecret, nil)
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

func (a *Azure) getAzureADGroupsClient() (graphrbac.GroupsClient, error) {
	groupsClient := graphrbac.NewGroupsClient(a.TenantID)
	authorizer, err := a.getGraphAuthorizer()
	if err != nil {
		return graphrbac.GroupsClient{}, err
	}

	groupsClient.Authorizer = authorizer

	return groupsClient, nil
}

func (a *Azure) getAzureADUsersClient() (graphrbac.UsersClient, error) {
	usersClient := graphrbac.NewUsersClient(a.TenantID)
	authorizer, err := a.getGraphAuthorizer()
	if err != nil {
		return graphrbac.UsersClient{}, err
	}

	usersClient.Authorizer = authorizer

	return usersClient, nil
}

// StartSyncTickerAzureADGroups initiates a ticker that will sync Azure AD Groups
func (a *Azure) StartSyncTickerAzureADGroups(syncInterval time.Duration) (*time.Ticker, chan bool, error) {
	log := logr.FromContext(a.Context)

	ticker := time.NewTicker(syncInterval)
	syncChan := make(chan bool)

	err := a.syncAzureADGroupsCache("initial")
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
				_ = a.syncAzureADGroupsCache("ticker")
			}
		}
	}()

	return ticker, syncChan, nil
}

func (a *Azure) syncAzureADGroupsCache(syncReason string) error {
	log := logr.FromContext(a.Context)

	groups, err := a.getAllGroups()
	if err != nil {
		log.Error(err, "Unable to syncronize groups")
		return err
	}

	for _, group := range groups {
		_, found, err := a.Cache.GetGroup(*group.ObjectID)
		if err != nil {
			return err
		}
		if !found {
			a.Cache.SetGroup(*group.ObjectID, models.Group{Name: *group.DisplayName})
		}
	}

	log.Info("Synchronized Azure AD groups to cache", "groupCount", len(groups), "syncReason", syncReason)

	return nil
}
