package user

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/coreos/go-oidc"
	"github.com/patrickmn/go-cache"
	"github.com/xenitab/azad-kube-proxy/pkg/azure"
	"github.com/xenitab/azad-kube-proxy/pkg/claims"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

// User is the struct for a currently logged in user
type User struct {
	Username string
	Groups   []string
}

// GetUser returns the user or an error
func (u User) GetUser(ctx context.Context, config config.Config, usersClient graphrbac.UsersClient, cache *cache.Cache, token *oidc.IDToken) (User, error) {
	username, err := u.getUsername(token)
	if err != nil {
		return User{}, err
	}

	objectID, err := u.getObjectID(token)
	if err != nil {
		return User{}, err
	}

	groups, err := u.getGroups(ctx, objectID, config, usersClient, cache)
	if err != nil {
		return User{}, err
	}

	user := User{
		Username: username,
		Groups:   groups,
	}

	return user, nil
}

func (u User) getUsername(token *oidc.IDToken) (string, error) {
	var tokenClaims claims.AzureClaims

	if err := token.Claims(&tokenClaims); err != nil {
		return "", err
	}

	return tokenClaims.Username, nil
}

func (u User) getObjectID(token *oidc.IDToken) (string, error) {
	var tokenClaims claims.AzureClaims

	if err := token.Claims(&tokenClaims); err != nil {
		return "", err
	}

	return tokenClaims.ObjectID, nil
}

func (u User) getGroups(ctx context.Context, objectID string, config config.Config, usersClient graphrbac.UsersClient, cache *cache.Cache) ([]string, error) {
	groups, err := azure.GetAzureADGroupNamesFromCache(ctx, objectID, config, usersClient, cache)
	if err != nil {
		return nil, err
	}

	return groups, nil
}
