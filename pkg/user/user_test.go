package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type fakeAzureClient struct {
	fakeError error
}

// GetUserGroups ...
func (client *fakeAzureClient) GetUserGroups(ctx context.Context, objectID string, userType models.UserType) ([]models.Group, error) {
	return nil, client.fakeError
}

// StartSyncGroups ...
func (client *fakeAzureClient) StartSyncGroups(ctx context.Context, syncInterval time.Duration) (*time.Ticker, chan bool, error) {
	return nil, nil, nil
}

// Valid ...
func (client *fakeAzureClient) Valid(ctx context.Context) bool {
	return true
}

func TestGetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	config := config.Config{}
	azureClient := &fakeAzureClient{
		fakeError: nil,
	}
	azureClientError := &fakeAzureClient{
		fakeError: errors.New("Fake error"),
	}

	cases := []struct {
		userClient          ClientInterface
		username            string
		objectID            string
		expectedUserType    models.UserType
		expectedErrContains string
	}{
		{
			userClient:          NewUserClient(config, azureClient),
			username:            "",
			objectID:            "00000000-0000-0000-0000-000000000000",
			expectedUserType:    models.ServicePrincipalUserType,
			expectedErrContains: "",
		},
		{
			userClient:          NewUserClient(config, azureClient),
			username:            "username",
			objectID:            "00000000-0000-0000-0000-000000000000",
			expectedUserType:    models.NormalUserType,
			expectedErrContains: "",
		},
		{
			userClient:          NewUserClient(config, azureClientError),
			username:            "username",
			objectID:            "00000000-0000-0000-0000-000000000000",
			expectedUserType:    models.NormalUserType,
			expectedErrContains: "Fake error",
		},
	}

	for _, c := range cases {
		user, err := c.userClient.GetUser(ctx, c.username, c.objectID)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.expectedUserType, user.Type)
	}
}
