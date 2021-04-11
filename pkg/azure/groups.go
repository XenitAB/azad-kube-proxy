package azure

import (
	"context"

	"github.com/go-logr/logr"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type groups struct {
	cacheClient  cache.ClientInterface
	groupsClient *hamiltonMsgraph.GroupsClient
	graphFilter  string
}

func newGroups(ctx context.Context, cacheClient cache.ClientInterface, groupsClient *hamiltonMsgraph.GroupsClient, graphFilter string) *groups {
	return &groups{
		cacheClient:  cacheClient,
		groupsClient: groupsClient,
		graphFilter:  graphFilter,
	}
}

func (groups *groups) getAllGroups(ctx context.Context) (*[]hamiltonMsgraph.Group, error) {
	log := logr.FromContext(ctx)

	groupsResponse, responseCode, err := groups.groupsClient.List(ctx, groups.graphFilter)
	if err != nil {
		log.Error(err, "Unable to get groups", "responseCode", responseCode)
		return nil, err
	}

	return groupsResponse, nil
}

func (groups *groups) syncAzureADGroupsCache(ctx context.Context, syncReason string) error {
	log := logr.FromContext(ctx)

	groupsResponse, err := groups.getAllGroups(ctx)
	if err != nil {
		log.Error(err, "Unable to syncronize groups")
		return err
	}

	for _, group := range *groupsResponse {
		err := groups.cacheClient.SetGroup(ctx, *group.ID, models.Group{
			Name:     *group.DisplayName,
			ObjectID: *group.ID,
		})
		if err != nil {
			return err
		}
	}

	log.Info("Synchronized Azure AD groups to cache", "groupCount", len(*groupsResponse), "syncReason", syncReason)

	return nil
}
