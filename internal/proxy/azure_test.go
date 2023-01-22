package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func TestNewAzureClient(t *testing.T) {
	clientID := testGetEnvOrSkip(t, "CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.Discard())

	memCache, err := newMemoryCache(5 * time.Minute)
	require.NoError(t, err)

	cases := []struct {
		clientID            string
		clientSecret        string
		tenantID            string
		graphFilter         string
		cacheClient         cacheReadWriter
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
		_, err := newAzureClient(ctx, c.clientID, c.clientSecret, c.tenantID, c.graphFilter, c.cacheClient)
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

	memCache, err := newMemoryCache(5 * time.Minute)
	require.NoError(t, err)

	azureClient, err := newAzureClient(ctx, clientID, clientSecret, tenantID, graphFilter, memCache)
	require.NoError(t, err)

	cases := []struct {
		client      *azure
		expectedRes bool
	}{
		{
			client:      azureClient,
			expectedRes: true,
		},
	}

	for _, c := range cases {
		valid := c.client.valid(ctx)
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

	memCache, err := newMemoryCache(5 * time.Minute)
	require.NoError(t, err)

	azureClient, err := newAzureClient(ctx, clientID, clientSecret, tenantID, graphFilter, memCache)
	require.NoError(t, err)

	cases := []struct {
		objectID            string
		userModelType       userModelType
		expectedErrContains string
	}{
		{
			objectID:            userObjectID,
			userModelType:       normalUserModelType,
			expectedErrContains: "",
		},
		{
			objectID:            spObjectID,
			userModelType:       servicePrincipalUserModelType,
			expectedErrContains: "",
		},
		{
			objectID:            "",
			userModelType:       normalUserModelType,
			expectedErrContains: "unexpected status 404 with OData error: Request_ResourceNotFound:",
		},
		{
			objectID:            "",
			userModelType:       servicePrincipalUserModelType,
			expectedErrContains: "unexpected status 400 with OData error: Request_BadRequest:",
		},
	}

	for _, c := range cases {
		_, err := azureClient.getUserGroups(ctx, c.objectID, c.userModelType)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}
		require.NoError(t, err)
	}

	emptyAzureClient := azure{}
	_, err = emptyAzureClient.getUserGroups(ctx, "", "FAKE")
	require.ErrorContains(t, err, "Unknown userType: FAKE")
}

func TestStartSyncGroups(t *testing.T) {
	clientID := testGetEnvOrSkip(t, "CLIENT_ID")
	clientSecret := testGetEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := testGetEnvOrSkip(t, "TENANT_ID")
	graphFilter := ""
	ctx := logr.NewContext(context.Background(), logr.Discard())

	memCache, err := newMemoryCache(5 * time.Minute)
	require.NoError(t, err)

	azureClient, err := newAzureClient(ctx, clientID, clientSecret, tenantID, graphFilter, memCache)
	require.NoError(t, err)

	groupSyncTicker, groupSyncChan, err := azureClient.startSyncGroups(ctx, 1*time.Second)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)
	var stopGroupSync func() = func() {
		groupSyncTicker.Stop()
		groupSyncChan <- true
	}
	defer stopGroupSync()
}
