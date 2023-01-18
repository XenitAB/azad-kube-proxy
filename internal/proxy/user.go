package proxy

import (
	"context"
)

type User interface {
	GetUser(ctx context.Context, username, objectID string) (userModel, error)
}

type user struct {
	azure Azure

	cfg *Config
}

func newUser(cfg *Config, azureClient Azure) User {
	return &user{
		azure: azureClient,
		cfg:   cfg,
	}
}

func (u *user) GetUser(ctx context.Context, username, objectID string) (userModel, error) {
	userType := normalUserModelType
	if username == "" {
		username = objectID
		userType = servicePrincipalUserModelType
	}

	groups, err := u.azure.GetUserGroups(ctx, objectID, userType)
	if err != nil {
		return userModel{}, err
	}

	user := userModel{
		Username: username,
		ObjectID: objectID,
		Groups:   groups,
		Type:     userType,
	}

	return user, nil
}
