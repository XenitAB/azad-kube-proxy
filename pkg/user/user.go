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
	AzureClient azure.Client
}

// NewUserClient ...
func NewUserClient(ctx context.Context, config config.Config, c cache.Cache, a azure.Client) Client {
	return Client{
		Context:     ctx,
		Config:      config,
		Cache:       c,
		AzureClient: a,
	}
}

// GetUser returns the user or an error
func (client *Client) GetUser(ctx context.Context, username, objectID string, tokenGroups []string) (models.User, error) {
	userType := models.NormalUserType
	if username == "" {
		username = objectID
		userType = models.ServicePrincipalUserType
	}

	groups, err := client.getGroups(ctx, objectID, userType)
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

func (client *Client) getGroups(ctx context.Context, objectID string, userType models.UserType) ([]models.Group, error) {
	groups, err := client.AzureClient.GetUserGroupsFromCache(ctx, objectID, userType)
	if err != nil {
		return nil, err
	}

	return groups, nil
}
