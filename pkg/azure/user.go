package azure

import (
	"context"

	"github.com/go-logr/logr"
	hamiltonClients "github.com/manicminer/hamilton/clients"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type user struct {
	cacheClient cache.ClientInterface
	usersClient *hamiltonClients.UsersClient
}

func newUser(ctx context.Context, cacheClient cache.ClientInterface, usersClient *hamiltonClients.UsersClient) *user {
	return &user{
		cacheClient: cacheClient,
		usersClient: usersClient,
	}
}

func (user *user) getGroups(ctx context.Context, objectID string) ([]models.Group, error) {
	log := logr.FromContext(ctx)

	groupsResponse, responseCode, err := user.usersClient.ListGroupMemberships(ctx, objectID, "")
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for user", "objectID", objectID, "responseCode", responseCode)
		return nil, err
	}

	var groups []models.Group
	for _, group := range *groupsResponse {
		group, found, err := user.cacheClient.GetGroup(ctx, *group.ID)
		if err != nil {
			return nil, err
		}
		if found {
			groups = append(groups, group)
		}

	}

	return groups, nil
}
