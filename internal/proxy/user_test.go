package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func TestGetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cfg := &config{}
	azureClient := &testFakeAzureClient{
		fakeError: nil,
		t:         t,
	}
	azureClientError := &testFakeAzureClient{
		fakeError: errors.New("Fake error"),
		t:         t,
	}

	cases := []struct {
		userClient          User
		username            string
		objectID            string
		expectedUserType    userModelType
		expectedErrContains string
	}{
		{
			userClient:          newUser(cfg, azureClient),
			username:            "",
			objectID:            "00000000-0000-0000-0000-000000000000",
			expectedUserType:    servicePrincipalUserModelType,
			expectedErrContains: "",
		},
		{
			userClient:          newUser(cfg, azureClient),
			username:            "username",
			objectID:            "00000000-0000-0000-0000-000000000000",
			expectedUserType:    normalUserModelType,
			expectedErrContains: "",
		},
		{
			userClient:          newUser(cfg, azureClientError),
			username:            "username",
			objectID:            "00000000-0000-0000-0000-000000000000",
			expectedUserType:    normalUserModelType,
			expectedErrContains: "Fake error",
		},
	}

	for _, c := range cases {
		user, err := c.userClient.getUser(ctx, c.username, c.objectID)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.expectedUserType, user.Type)
	}
}

type testFakeAzureClient struct {
	fakeError error
	t         *testing.T
}

func (client *testFakeAzureClient) getUserGroups(ctx context.Context, objectID string, userType userModelType) ([]groupModel, error) {
	client.t.Helper()

	return nil, client.fakeError
}

func (client *testFakeAzureClient) startSyncGroups(ctx context.Context, syncInterval time.Duration) (*time.Ticker, chan bool, error) {
	client.t.Helper()
	return nil, nil, nil
}

func (client *testFakeAzureClient) valid(ctx context.Context) bool {
	client.t.Helper()
	return true
}
