package proxy

import (
	"context"

	"github.com/go-logr/logr"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
	hamiltonOdata "github.com/manicminer/hamilton/odata"
)

type azureUser struct {
	cache       cacheReadWriter
	usersClient *hamiltonMsgraph.UsersClient
}

func newAzureUser(ctx context.Context, cacheClient cacheReadWriter, usersClient *hamiltonMsgraph.UsersClient) *azureUser {
	return &azureUser{
		cache:       cacheClient,
		usersClient: usersClient,
	}
}

func (user *azureUser) getGroups(ctx context.Context, objectID string) ([]groupModel, error) {
	log := logr.FromContextOrDiscard(ctx)

	odataQuery := hamiltonOdata.Query{}

	groupsResponse, responseCode, err := user.usersClient.ListGroupMemberships(ctx, objectID, odataQuery)
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for user", "objectID", objectID, "responseCode", responseCode)
		return nil, err
	}

	var groups []groupModel
	for _, group := range *groupsResponse {
		group, found, err := user.cache.getGroup(ctx, *group.ID())
		if err != nil {
			return nil, err
		}
		if found {
			groups = append(groups, group)
		}

	}

	return groups, nil
}
