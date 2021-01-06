package user

import (
	"context"

	"github.com/coreos/go-oidc"
	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
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
func (u *User) GetUser(username, objectID string, tokenGroups []string) (models.User, error) {
	userType := models.NormalUserType
	if username == "" {
		username = objectID
		userType = models.ServicePrincipalUserType
	}

	groups, err := u.getGroups(objectID, userType)
	if err != nil {
		return models.User{}, err
	}

	user := models.User{
		Username: username,
		Groups:   groups,
		Type:     userType,
	}

	return user, nil
}

func (u *User) getGroups(objectID string, userType models.UserType) ([]models.Group, error) {
	groups, err := u.AzureClient.GetUserGroupsFromCache(objectID, userType)
	if err != nil {
		return nil, err
	}

	return groups, nil
}
