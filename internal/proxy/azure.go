package proxy

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	hamiltonAuth "github.com/manicminer/hamilton/auth"
	hamiltonEnvironments "github.com/manicminer/hamilton/environments"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
)

type AzureUser interface {
	getGroups(ctx context.Context, objectID string) ([]groupModel, error)
}

type Azure interface {
	GetUserGroups(ctx context.Context, objectID string, userType userModelType) ([]groupModel, error)
	StartSyncGroups(ctx context.Context, syncInterval time.Duration) (*time.Ticker, chan bool, error)
	Valid(ctx context.Context) bool
}

type azure struct {
	clientID             string
	clientSecret         string
	tenantID             string
	graphFilter          string
	cache                Cache
	groups               *azureGroups
	user                 *azureUser
	servicePrincipalUser *azureServicePrincipalUser
	authorizer           hamiltonAuth.Authorizer
}

func newAzureClient(ctx context.Context, clientID, clientSecret, tenantID, graphFilter string, cacheClient Cache) (*azure, error) {
	authConfig := &hamiltonAuth.Config{
		Environment:            hamiltonEnvironments.Global,
		TenantID:               tenantID,
		ClientID:               clientID,
		ClientSecret:           clientSecret,
		EnableClientSecretAuth: true,
	}

	authorizer, err := authConfig.NewAuthorizer(ctx, hamiltonEnvironments.MsGraphGlobal)
	if err != nil {
		return nil, err
	}

	usersClient := hamiltonMsgraph.NewUsersClient(tenantID)
	usersClient.BaseClient.Authorizer = authorizer
	usersClient.BaseClient.DisableRetries = true

	servicePrincipalsClient := hamiltonMsgraph.NewServicePrincipalsClient(tenantID)
	servicePrincipalsClient.BaseClient.Authorizer = authorizer
	servicePrincipalsClient.BaseClient.DisableRetries = true

	groupsClient := hamiltonMsgraph.NewGroupsClient(tenantID)
	groupsClient.BaseClient.Authorizer = authorizer
	groupsClient.BaseClient.DisableRetries = true

	if graphFilter != "" {
		graphFilter = fmt.Sprintf("startswith(displayName,'%s')", graphFilter)
	}

	return &azure{
		clientID:             clientID,
		clientSecret:         clientSecret,
		tenantID:             tenantID,
		graphFilter:          graphFilter,
		cache:                cacheClient,
		user:                 newAzureUser(ctx, cacheClient, usersClient),
		servicePrincipalUser: newServicePrincipalUser(ctx, cacheClient, servicePrincipalsClient),
		groups:               newGroups(ctx, cacheClient, groupsClient, graphFilter),
		authorizer:           authorizer,
	}, nil
}

func (client *azure) Valid(ctx context.Context) bool {
	log := logr.FromContextOrDiscard(ctx)
	token, err := client.authorizer.Token()

	if err != nil {
		log.Error(err, "Unable to get token from authorizer")
		return false
	}

	if !token.Valid() {
		log.Error(fmt.Errorf("Token not valid"), "Token not valid", "access_token", token.AccessToken)
		return false
	}

	if time.Now().After(token.Expiry) {
		log.Error(fmt.Errorf("Token expired"), "Token expired", "expiry", token.Expiry)
		return false
	}

	return true
}

// GetUserGroups ...
func (client *azure) GetUserGroups(ctx context.Context, objectID string, userType userModelType) ([]groupModel, error) {
	var user AzureUser

	switch userType {
	case normalUserModelType:
		user = client.user
	case servicePrincipalUserModelType:
		user = client.servicePrincipalUser
	default:
		return nil, fmt.Errorf("Unknown userType: %s", userType)
	}

	return user.getGroups(ctx, objectID)
}

// StartSyncGroups initiates a ticker that will sync Azure AD Groups
func (client *azure) StartSyncGroups(ctx context.Context, syncInterval time.Duration) (*time.Ticker, chan bool, error) {
	log := logr.FromContextOrDiscard(ctx)

	ticker := time.NewTicker(syncInterval)
	syncChan := make(chan bool)

	err := client.groups.syncAzureADGroupsCache(ctx, "initial")
	if err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			select {
			case <-syncChan:
				log.Info("Stopped StartSyncTickerAzureADGroups")
				return
			case <-ticker.C:
				_ = client.groups.syncAzureADGroupsCache(ctx, "ticker")
			}
		}
	}()

	return ticker, syncChan, nil
}
