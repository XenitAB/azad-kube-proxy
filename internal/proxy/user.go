package proxy

import (
	"context"

	"github.com/xenitab/azad-kube-proxy/internal/config"
	"github.com/xenitab/azad-kube-proxy/internal/models"
)

type User interface {
	GetUser(ctx context.Context, username, objectID string) (models.User, error)
}

type Client struct {
	azure Azure

	cfg *config.Config
}

func newUserClient(cfg *config.Config, azureClient Azure) User {
	return &Client{
		azure: azureClient,
		cfg:   cfg,
	}
}

func (client *Client) GetUser(ctx context.Context, username, objectID string) (models.User, error) {
	userType := models.NormalUserType
	if username == "" {
		username = objectID
		userType = models.ServicePrincipalUserType
	}

	groups, err := client.azure.GetUserGroups(ctx, objectID, userType)
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
