package proxy

import (
	"context"
)

type User interface {
	getUser(ctx context.Context, username, objectID string) (userModel, error)
}

type user struct {
	azure Azure

	cfg *config
}

func newUser(cfg *config, azureClient Azure) User {
	return &user{
		azure: azureClient,
		cfg:   cfg,
	}
}

func (u *user) getUser(ctx context.Context, username, objectID string) (userModel, error) {
	userType := normalUserModelType
	if username == "" {
		username = objectID
		userType = servicePrincipalUserModelType
	}

	groups, err := u.azure.getUserGroups(ctx, objectID, userType)
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
