package user

import (
	"context"

	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// Client ...
type Client struct {
	Context     context.Context
	Config      config.Config
	Cache       cache.Cache
	AzureClient *azure.Client
}

// NewUserClient ...
func NewUserClient(config config.Config, azureClient *azure.Client) *Client {
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
