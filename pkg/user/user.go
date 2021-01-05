package user

import (
	"context"

	"github.com/coreos/go-oidc"
	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/claims"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// Client ...
type Client interface {
	GetUser(token *oidc.IDToken) (models.User, error)
}

// User ...
type User struct {
	Context     context.Context
	Config      config.Config
	Cache       cache.Client
	AzureClient azure.Azure
}

// NewUserClient ...
func NewUserClient(ctx context.Context, config config.Config, c cache.Client, a azure.Azure) User {
	return User{
		Context:     ctx,
		Config:      config,
		Cache:       c,
		AzureClient: a,
	}
}

// GetUser returns the user or an error
func (u *User) GetUser(token *oidc.IDToken) (models.User, error) {
	username, err := u.getUsername(token)
	if err != nil {
		return models.User{}, err
	}

	objectID, err := u.getObjectID(token)
	if err != nil {
		return models.User{}, err
	}

	groups, err := u.getGroups(objectID)
	if err != nil {
		return models.User{}, err
	}

	user := models.User{
		Username: username,
		Groups:   groups,
	}

	return user, nil
}

func (u *User) getUsername(token *oidc.IDToken) (string, error) {
	var tokenClaims claims.AzureClaims

	if err := token.Claims(&tokenClaims); err != nil {
		return "", err
	}

	return tokenClaims.Username, nil
}

func (u *User) getObjectID(token *oidc.IDToken) (string, error) {
	var tokenClaims claims.AzureClaims

	if err := token.Claims(&tokenClaims); err != nil {
		return "", err
	}

	return tokenClaims.ObjectID, nil
}

func (u *User) getGroups(objectID string) ([]models.Group, error) {
	groups, err := u.AzureClient.GetUserGroupsFromCache(objectID)
	if err != nil {
		return nil, err
	}

	return groups, nil
}
