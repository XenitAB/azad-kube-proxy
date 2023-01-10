package azure

import (
	"context"

	"github.com/go-logr/logr"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
	hamiltonOdata "github.com/manicminer/hamilton/odata"
	"github.com/xenitab/azad-kube-proxy/internal/cache"
	"github.com/xenitab/azad-kube-proxy/internal/models"
)

type user struct {
	cacheClient cache.ClientInterface
	usersClient *hamiltonMsgraph.UsersClient
}

func newUser(ctx context.Context, cacheClient cache.ClientInterface, usersClient *hamiltonMsgraph.UsersClient) *user {
	return &user{
		cacheClient: cacheClient,
		usersClient: usersClient,
	}
}

func (user *user) getGroups(ctx context.Context, objectID string) ([]models.Group, error) {
	log := logr.FromContextOrDiscard(ctx)

	odataQuery := hamiltonOdata.Query{}

	groupsResponse, responseCode, err := user.usersClient.ListGroupMemberships(ctx, objectID, odataQuery)
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for user", "objectID", objectID, "responseCode", responseCode)
		return nil, err
	}

	var groups []models.Group
	for _, group := range *groupsResponse {
		group, found, err := user.cacheClient.GetGroup(ctx, *group.ID())
		if err != nil {
			return nil, err
		}
		if found {
			groups = append(groups, group)
		}

	}

	return groups, nil
}
