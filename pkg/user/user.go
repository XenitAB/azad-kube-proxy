package user

import (
	"context"

	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// ClientInterface ...
type ClientInterface interface {
	GetUser(ctx context.Context, username, objectID string) (models.User, error)
}

// Client ...
type Client struct {
	Config      config.Config
	Cache       *cache.ClientInterface
	AzureClient azure.ClientInterface
}

// NewUserClient ...
func NewUserClient(config config.Config, azureClient azure.ClientInterface) ClientInterface {
	return &Client{
		Config:      config,
		AzureClient: azureClient,
	}
}

// GetUser returns the user or an error
func (client *Client) GetUser(ctx context.Context, username, objectID string) (models.User, error) {
	userType := models.NormalUserType
	if username == "" {
		username = objectID
		userType = models.ServicePrincipalUserType
	}

	groups, err := client.AzureClient.GetUserGroups(ctx, objectID, userType)
	if err != nil {
		return models.User{}, err
	}

	user := models.User{
		Username: username,
		ObjectID: objectID,
		Groups:   groups,
		Type:     userType,
	}

	return user, nil
}
