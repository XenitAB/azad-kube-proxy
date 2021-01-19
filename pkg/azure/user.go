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

func newUser(ctx context.Context, cacheClient cache.ClientInterface, usersClient *hamiltonClients.UsersClient) (*user, error) {
	user := &user{
		cacheClient: cacheClient,
		usersClient: usersClient,
	}

	return user, nil
}

func (user *user) getGroups(ctx context.Context, objectID string) ([]models.Group, error) {
	var groupIDs []string
	var err error

	groupIDs, err = user.getUserGroups(ctx, objectID)
	if err != nil {
		return nil, err
	}

	var groupNames []models.Group
	for _, groupID := range groupIDs {
		group, found, err := user.cacheClient.GetGroup(ctx, groupID)
		if err != nil {
			return nil, err
		}
		if found {
			groupNames = append(groupNames, group)
		}
	}

	return groupNames, nil
}

func (user *user) getUserGroups(ctx context.Context, objectID string) ([]string, error) {
	log := logr.FromContext(ctx)

	groupsResponse, responseCode, err := user.usersClient.ListGroupMemberships(ctx, objectID, "")
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for user", "objectID", objectID, "responseCode", responseCode)
		return nil, err
	}

	var groups []string
	for _, group := range *groupsResponse {
		groups = append(groups, *group.ID)
	}

	return groups, nil
}
