package azure

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/to"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/go-logr/logr"
	"github.com/jongio/azidext/go/azidext"
	"github.com/patrickmn/go-cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

// GetAzureADGroupNamesFromCache returns the group names the user is a member of
func GetAzureADGroupNamesFromCache(ctx context.Context, objectID string, config config.Config, usersClient graphrbac.UsersClient, cache *cache.Cache) ([]string, error) {
	groupIDs, err := GetUserAzureADGroups(ctx, objectID, config, usersClient)
	if err != nil {
		return nil, err
	}

	var groupNames []string
	for _, groupID := range groupIDs {
		cacheResponse, found := cache.Get(groupID)
		if found {
			groupNames = append(groupNames, cacheResponse.(string))
		}
	}

	return groupNames, nil
}

// GetUserAzureADGroups returns the groups the user is a member of
func GetUserAzureADGroups(ctx context.Context, objectID string, config config.Config, usersClient graphrbac.UsersClient) ([]string, error) {
	log := logr.FromContext(ctx)

	groupsResponse, err := usersClient.GetMemberGroups(ctx, objectID, graphrbac.UserGetMemberGroupsParameters{
		SecurityEnabledOnly: to.BoolPtr(false),
	})
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups from user", "UserObjectID", objectID)
		return nil, err
	}

	return *groupsResponse.Value, nil
}

// GetAzureADGroups returns the groups in Azure AD
func GetAzureADGroups(ctx context.Context, config config.Config, groupsClient graphrbac.GroupsClient, graphFilter string) ([]graphrbac.ADGroup, error) {
	log := logr.FromContext(ctx)

	var groups []graphrbac.ADGroup
	for list, err := groupsClient.List(ctx, graphFilter); list.NotDone(); err = list.NextWithContext(ctx) {
		if err != nil {
			log.Error(err, "Unable to list Azure AD groups", "graphFilter", graphFilter)
			return nil, err
		}
		for _, group := range list.Values() {
			groups = append(groups, group)
		}
	}

	return groups, nil
}

func getGraphAuthorizer(ctx context.Context, config config.Config) (autorest.Authorizer, error) {
	log := logr.FromContext(ctx)

	cred, err := azidentity.NewClientSecretCredential(config.TenantID, config.ClientID, config.ClientSecret, nil)
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

// GetAzureADGroupsClient returns an Azure graphrbac.GroupsClient or error
func GetAzureADGroupsClient(ctx context.Context, config config.Config) (graphrbac.GroupsClient, error) {
	groupsClient := graphrbac.NewGroupsClient(config.TenantID)
	authorizer, err := getGraphAuthorizer(ctx, config)
	if err != nil {
		return graphrbac.GroupsClient{}, err
	}

	groupsClient.Authorizer = authorizer

	return groupsClient, nil
}

// GetAzureADUsersClient returns an Azure graphrbac.UsersClient or error
func GetAzureADUsersClient(ctx context.Context, config config.Config) (graphrbac.UsersClient, error) {
	usersClient := graphrbac.NewUsersClient(config.TenantID)
	authorizer, err := getGraphAuthorizer(ctx, config)
	if err != nil {
		return graphrbac.UsersClient{}, err
	}

	usersClient.Authorizer = authorizer

	return usersClient, nil
}

// SyncTickerAzureADGroups initiates a ticker that will sync Azure AD Groups
func SyncTickerAzureADGroups(ctx context.Context, config config.Config, groupsClient graphrbac.GroupsClient, graphFilter string, syncInterval time.Duration, cache *cache.Cache) (*time.Ticker, chan bool, error) {
	log := logr.FromContext(ctx)

	ticker := time.NewTicker(syncInterval)
	syncChan := make(chan bool)

	err := syncAzureADGroupsCache(ctx, config, groupsClient, graphFilter, cache, "initial")
	if err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			select {
			case <-syncChan:
				log.Info("Stopped SyncTickerAzureADGroups")
				return
			case _ = <-ticker.C:
				_ = syncAzureADGroupsCache(ctx, config, groupsClient, graphFilter, cache, "ticker")
			}
		}
	}()

	return ticker, syncChan, nil
}

func syncAzureADGroupsCache(ctx context.Context, config config.Config, groupsClient graphrbac.GroupsClient, graphFilter string, cache *cache.Cache, syncReason string) error {
	log := logr.FromContext(ctx)

	groups, err := GetAzureADGroups(ctx, config, groupsClient, graphFilter)
	if err != nil {
		log.Error(err, "Unable to syncronize groups")
		return err
	}

	for _, group := range groups {
		_, found := cache.Get(*group.ObjectID)
		if !found {
			cache.Set(*group.ObjectID, *group.DisplayName, 5*time.Minute)
		}
	}

	log.Info("Synchronized Azure AD groups to cache", "groupCount", len(groups), "syncReason", syncReason)

	return nil
}
