package azure

import (
	"context"

	"github.com/go-logr/logr"
	hamiltonMsgraph "github.com/manicminer/hamilton/msgraph"
	hamiltonOdata "github.com/manicminer/hamilton/odata"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type servicePrincipalUser struct {
	cacheClient             cache.ClientInterface
	servicePrincipalsClient *hamiltonMsgraph.ServicePrincipalsClient
}

func newServicePrincipalUser(ctx context.Context, cacheClient cache.ClientInterface, servicePrincipalsClient *hamiltonMsgraph.ServicePrincipalsClient) *servicePrincipalUser {
	return &servicePrincipalUser{
		cacheClient:             cacheClient,
		servicePrincipalsClient: servicePrincipalsClient,
	}
}

func (user *servicePrincipalUser) getGroups(ctx context.Context, objectID string) ([]models.Group, error) {
	log := logr.FromContext(ctx)

	odataQuery := hamiltonOdata.Query{}

	groupsResponse, responseCode, err := user.servicePrincipalsClient.ListGroupMemberships(ctx, objectID, odataQuery)
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for service principal", "objectID", objectID, "responseCode", responseCode)
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
