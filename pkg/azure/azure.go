package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/manicminer/hamilton/auth"
	hamiltonAuth "github.com/manicminer/hamilton/auth"
	hamiltonClients "github.com/manicminer/hamilton/clients"
	hamiltonEnvironments "github.com/manicminer/hamilton/environments"
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
	Valid(ctx context.Context) bool
}

// Client ...
type Client struct {
	clientID             string
	clientSecret         string
	tenantID             string
	graphFilter          string
	cacheClient          cache.ClientInterface
	groups               *groups
	user                 *user
	servicePrincipalUser *servicePrincipalUser
	authorizer           hamiltonAuth.Authorizer
}

// NewAzureClient returns an Azure client or error
func NewAzureClient(ctx context.Context, clientID, clientSecret, tenantID, graphFilter string, cacheClient cache.ClientInterface) (*Client, error) {
	authConfig := &auth.Config{
		Environment:            hamiltonEnvironments.Global,
		TenantID:               tenantID,
		ClientID:               clientID,
		ClientSecret:           clientSecret,
		EnableClientSecretAuth: true,
	}

	authorizer, err := authConfig.NewAuthorizer(ctx, hamiltonAuth.MsGraph)
	if err != nil {
		return nil, err
	}

	usersClient := hamiltonClients.NewUsersClient(tenantID)
	usersClient.BaseClient.Authorizer = authorizer

	servicePrincipalsClient := hamiltonClients.NewServicePrincipalsClient(tenantID)
	servicePrincipalsClient.BaseClient.Authorizer = authorizer

	groupsClient := hamiltonClients.NewGroupsClient(tenantID)
	groupsClient.BaseClient.Authorizer = authorizer

	if graphFilter != "" {
		graphFilter = fmt.Sprintf("startswith(displayName,'%s')", graphFilter)
	}

	return &Client{
		clientID:             clientID,
		clientSecret:         clientSecret,
		tenantID:             tenantID,
		graphFilter:          graphFilter,
		cacheClient:          cacheClient,
		user:                 newUser(ctx, cacheClient, usersClient),
		servicePrincipalUser: newServicePrincipalUser(ctx, cacheClient, servicePrincipalsClient),
		groups:               newGroups(ctx, cacheClient, groupsClient, graphFilter),
		authorizer:           authorizer,
	}, nil
}

func (client *Client) Valid(ctx context.Context) bool {
	log := logr.FromContext(ctx)
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
