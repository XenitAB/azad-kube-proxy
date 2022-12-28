package azure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

func TestNewAzureClient(t *testing.T) {
	clientID := testGetEnvOrSkip(t, "CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.Discard())

	memCache, err := cache.NewCache(ctx, models.MemoryCacheEngine, config.Config{})
	require.NoError(t, err)

	cases := []struct {
		clientID            string
		clientSecret        string
		tenantID            string
		graphFilter         string
		cacheClient         cache.ClientInterface
		expectedErrContains string
	}{
		{
			clientID:            clientID,
			clientSecret:        clientSecret,
			tenantID:            tenantID,
			graphFilter:         "",
			cacheClient:         memCache,
			expectedErrContains: "",
		},
		{
			clientID:            clientID,
			clientSecret:        clientSecret,
			tenantID:            tenantID,
			graphFilter:         "prefix",
			cacheClient:         memCache,
			expectedErrContains: "",
		},
		{
			clientID:            clientID,
			clientSecret:        clientSecret,
			tenantID:            "",
			graphFilter:         "",
			cacheClient:         memCache,
			expectedErrContains: "no Authorizer could be configured, please check your configuration",
		},
		{
			clientID:            "",
			clientSecret:        "",
			tenantID:            tenantID,
			graphFilter:         "",
			cacheClient:         memCache,
			expectedErrContains: "no Authorizer could be configured, please check your configuration",
		},
	}

	for _, c := range cases {
		_, err := NewAzureClient(ctx, c.clientID, c.clientSecret, c.tenantID, c.graphFilter, c.cacheClient)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}
		require.NoError(t, err)
	}
}

func TestValid(t *testing.T) {
	clientID := testGetEnvOrSkip(t, "CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	graphFilter := ""
	ctx := logr.NewContext(context.Background(), logr.Discard())

	memCache, err := cache.NewCache(ctx, models.MemoryCacheEngine, config.Config{})
	require.NoError(t, err)

	azureClient, err := NewAzureClient(ctx, clientID, clientSecret, tenantID, graphFilter, memCache)
	require.NoError(t, err)

	cases := []struct {
		client      *Client
		expectedRes bool
	}{
		{
			client:      azureClient,
			expectedRes: true,
		},
	}

	for _, c := range cases {
		valid := c.client.Valid(ctx)
		require.True(t, valid)
	}
}

func TestGetUserGroups(t *testing.T) {
	clientID := testGetEnvOrSkip(t, "CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	userObjectID := testGetEnvOrSkip(t, "TEST_USER_OBJECT_ID")
	spObjectID := testGetEnvOrSkip(t, "TEST_USER_SP_OBJECT_ID")
	graphFilter := ""
	ctx := logr.NewContext(context.Background(), logr.Discard())

	memCache, err := cache.NewCache(ctx, models.MemoryCacheEngine, config.Config{})
	require.NoError(t, err)

	azureClient, err := NewAzureClient(ctx, clientID, clientSecret, tenantID, graphFilter, memCache)
	require.NoError(t, err)

	cases := []struct {
		objectID            string
		userType            models.UserType
		expectedErrContains string
	}{
		{
			objectID:            userObjectID,
			userType:            models.NormalUserType,
			expectedErrContains: "",
		},
		{
			objectID:            spObjectID,
			userType:            models.ServicePrincipalUserType,
			expectedErrContains: "",
		},
		{
			objectID:            "",
			userType:            models.NormalUserType,
			expectedErrContains: "unexpected status 404 with OData error: Request_ResourceNotFound:",
		},
		{
			objectID:            "",
			userType:            models.ServicePrincipalUserType,
			expectedErrContains: "unexpected status 400 with OData error: Request_BadRequest:",
		},
	}

	for _, c := range cases {
		_, err := azureClient.GetUserGroups(ctx, c.objectID, c.userType)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}
		require.NoError(t, err)
	}

	emptyAzureClient := Client{}
	_, err = emptyAzureClient.GetUserGroups(ctx, "", "FAKE")
	require.ErrorContains(t, err, "Unknown userType: FAKE")
}

func TestStartSyncGroups(t *testing.T) {
	clientID := testGetEnvOrSkip(t, "CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	graphFilter := ""
	ctx := logr.NewContext(context.Background(), logr.Discard())

	memCache, err := cache.NewCache(ctx, models.MemoryCacheEngine, config.Config{})
	require.NoError(t, err)

	azureClient, err := NewAzureClient(ctx, clientID, clientSecret, tenantID, graphFilter, memCache)
	require.NoError(t, err)

	groupSyncTicker, groupSyncChan, err := azureClient.StartSyncGroups(ctx, 1*time.Second)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)
	var stopGroupSync func() = func() {
		groupSyncTicker.Stop()
		groupSyncChan <- true
	}
	defer stopGroupSync()
}

func testGetEnvOrSkip(t *testing.T, envVar string) string {
	t.Helper()

	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s environment variable is empty, skipping.", envVar)
	}

	return v
}
