package azure

import (
	"context"

	"github.com/go-logr/logr"
	hamiltonClients "github.com/manicminer/hamilton/clients"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type servicePrincipalUser struct {
	cacheClient             cache.ClientInterface
	servicePrincipalsClient *hamiltonClients.ServicePrincipalsClient
}

func newServicePrincipalUser(ctx context.Context, cacheClient cache.ClientInterface, servicePrincipalsClient *hamiltonClients.ServicePrincipalsClient) (*servicePrincipalUser, error) {
	return &servicePrincipalUser{
		cacheClient:             cacheClient,
		servicePrincipalsClient: servicePrincipalsClient,
	}, nil
}

func (user *servicePrincipalUser) getGroups(ctx context.Context, objectID string) ([]models.Group, error) {
	var groupIDs []string
	var err error

	groupIDs, err = user.getServicePrincipalGroups(ctx, objectID)
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

func (user *servicePrincipalUser) getServicePrincipalGroups(ctx context.Context, objectID string) ([]string, error) {
	log := logr.FromContext(ctx)

	groupsResponse, responseCode, err := user.servicePrincipalsClient.ListGroupMemberships(ctx, objectID, "")
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for service principal", "objectID", objectID, "responseCode", responseCode)
		return nil, err
	}

	var groups []string
	for _, group := range *groupsResponse {
		groups = append(groups, *group.ID)
	}

	return groups, nil
}
