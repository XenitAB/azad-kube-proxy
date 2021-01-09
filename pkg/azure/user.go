package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/to"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type user struct {
	cache       cache.Cache
	usersClient graphrbac.UsersClient
}

func newUser(ctx context.Context, cache cache.Cache, usersClient graphrbac.UsersClient) (*user, error) {
	user := &user{
		cache:       cache,
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
		group, found, err := user.cache.GetGroup(ctx, groupID)
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

	groupsResponse, err := user.usersClient.GetMemberGroups(ctx, objectID, graphrbac.UserGetMemberGroupsParameters{
		SecurityEnabledOnly: to.BoolPtr(false),
	})
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for user", "objectID", objectID)
		return nil, err
	}

	return *groupsResponse.Value, nil
}
