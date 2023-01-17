package proxy

import (
	"context"

	"github.com/xenitab/azad-kube-proxy/internal/config"
	"github.com/xenitab/azad-kube-proxy/internal/models"
)

type User interface {
	GetUser(ctx context.Context, username, objectID string) (models.User, error)
}

type user struct {
	azure Azure

	cfg *config.Config
}

func newUser(cfg *config.Config, azureClient Azure) User {
	return &user{
		azure: azureClient,
		cfg:   cfg,
	}
}

func (u *user) GetUser(ctx context.Context, username, objectID string) (models.User, error) {
	userType := models.NormalUserType
	if username == "" {
		username = objectID
		userType = models.ServicePrincipalUserType
	}

	groups, err := u.azure.GetUserGroups(ctx, objectID, userType)
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
