package user

import (
	"context"
	"testing"
	"time"

	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type fakeAzureClient struct{}

// GetUserGroups ...
func (client *fakeAzureClient) GetUserGroups(ctx context.Context, objectID string, userType models.UserType) ([]models.Group, error) {
	return nil, nil
}

// StartSyncGroups ...
func (client *fakeAzureClient) StartSyncGroups(ctx context.Context, syncInterval time.Duration) (*time.Ticker, chan bool, error) {
	return nil, nil, nil
}

func TestGetUser(t *testing.T) {
	config := config.Config{}
	azureClient := &fakeAzureClient{}

	cases := []struct {
		userClient       *Client
		username         string
		objectID         string
		expectedUserType models.UserType
	}{
		{
			userClient:       NewUserClient(config, azureClient),
			username:         "",
			objectID:         "00000000-0000-0000-0000-000000000000",
			expectedUserType: models.ServicePrincipalUserType,
		},
		{
			userClient:       NewUserClient(config, azureClient),
			username:         "username",
			objectID:         "00000000-0000-0000-0000-000000000000",
			expectedUserType: models.NormalUserType,
		},
	}

	for _, c := range cases {
		ctx := context.Background()
		user, err := c.userClient.GetUser(ctx, c.username, c.objectID)

		if user.Type != c.expectedUserType {
			t.Errorf("Expected user type (%s) was not returned: %s", c.expectedUserType, user.Type)
		}

		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}
	}
}
