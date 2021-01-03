package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/pkg/claims"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

// GetAzureADGroups return the groups the user is a member of
func GetAzureADGroups(ctx context.Context, tokenClaims claims.AzureClaims, config config.Config) ([]string, error) {
	log := logr.FromContext(ctx)

	type distributedGroupsPost struct {
		SecurityEnabledOnly bool `json:"securityEnabledOnly"`
	}

	body := &distributedGroupsPost{
		SecurityEnabledOnly: false,
	}

	payloadBuf := new(bytes.Buffer)
	json.NewEncoder(payloadBuf).Encode(body)
	graphEndpoint := fmt.Sprintf("%s?api-version=1.6", tokenClaims.ClaimSources.Source1.Endpoint)
	distributedGroupsReq, err := http.NewRequest("POST", graphEndpoint, payloadBuf)
	if err != nil {
		log.Error(err, "Unable to create distributed claims request")
		return []string{""}, err
	}

	distributedGroupsReq.Header.Set("Content-Type", "application/json")

	token, err := getAzureToken(ctx, config)
	if err != nil {
		return []string{""}, err
	}

	distributedGroupsReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Token))

	distributedGroupsClient := &http.Client{}
	distributedGroupsRes, err := distributedGroupsClient.Do(distributedGroupsReq)
	if err != nil {
		log.Error(err, "Unable to fetch distributed claims")
		return []string{""}, err
	}

	defer distributedGroupsRes.Body.Close()

	decoder := json.NewDecoder(distributedGroupsRes.Body)
	var responseData struct {
		Value []string `json:"value"`
	}
	err = decoder.Decode(&responseData)

	return responseData.Value, nil
}

func getAzureToken(ctx context.Context, config config.Config) (*azcore.AccessToken, error) {
	log := logr.FromContext(ctx)

	cred, err := azidentity.NewClientSecretCredential(config.TenantID, config.ClientID, config.ClientSecret, nil)
	if err != nil {
		log.Error(err, "azidentity.NewClientSecretCredential")
		return nil, err
	}

	token, err := cred.GetToken(ctx, azcore.TokenRequestOptions{Scopes: []string{"https://graph.windows.net/.default"}})
	if err != nil {
		log.Error(err, "Unable to get token")
		return nil, err
	}

	return token, nil
}
