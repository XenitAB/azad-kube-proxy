package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type servicePrincipalUser struct {
	azureCredential *azidentity.ClientSecretCredential
	cache           cache.Cache
	msGraphToken    *azcore.AccessToken
}

func newServicePrincipalUser(ctx context.Context, azureCredential *azidentity.ClientSecretCredential, cache cache.Cache) (*servicePrincipalUser, error) {
	user := &servicePrincipalUser{
		azureCredential: azureCredential,
		cache:           cache,
	}

	var err error
	user.msGraphToken, err = user.getMSGraphToken(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
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

func (user *servicePrincipalUser) getServicePrincipalGroups(ctx context.Context, objectID string) ([]string, error) {
	log := logr.FromContext(ctx)

	url, err := url.Parse(fmt.Sprintf("https://graph.microsoft.com/v1.0/servicePrincipals/%s/transitiveMemberOf/microsoft.graph.group?$select=id", objectID))
	if err != nil {
		log.Error(err, "Unable to parse URL")
		return nil, err
	}

	msGraphToken, err := user.getMSGraphToken(ctx)
	if err != nil {
		log.Error(err, "Unable to get MS Graph Token")
		return nil, err
	}

	req := &http.Request{
		Method: "GET",
		URL:    url,
		Header: map[string][]string{
			"Authorization": {fmt.Sprintf("Bearer %s", msGraphToken.Token)},
			"Content-type":  {"application/json"},
		},
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err, "Unable to get Azure AD groups for service principal", "objectID", objectID)
		return nil, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Error(err, "Unable to read body of response", "objectID", objectID)
		return nil, err
	}

	type responseValue struct {
		ID string `json:"id"`
	}

	var responseData struct {
		Values []responseValue `json:"value"`
	}

	json.Unmarshal(body, &responseData)

	var groups []string
	for _, group := range responseData.Values {
		groups = append(groups, group.ID)
	}

	return groups, nil
}

// GetMSGraphToken generates a new token if none exists and an existing if it not expired
func (user *servicePrincipalUser) getMSGraphToken(ctx context.Context) (*azcore.AccessToken, error) {
	log := logr.FromContext(ctx)

	generateNewToken := true
	if user.msGraphToken != nil {
		if user.msGraphToken.ExpiresOn.After(time.Now().Add(-5 * time.Minute)) {
			generateNewToken = false
		}
	}

	if generateNewToken {
		token, err := user.azureCredential.GetToken(ctx, azcore.TokenRequestOptions{Scopes: []string{"https://graph.microsoft.com/.default"}})
		if err != nil {
			log.Error(err, "client.AzureCredential.GetToken")
			return nil, err
		}

		return token, nil
	}

	return user.msGraphToken, nil
}
