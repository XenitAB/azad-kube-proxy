package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
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
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	config := config.Config{}
	azureClient := &fakeAzureClient{
		fakeError: nil,
	}
	azureClientError := &fakeAzureClient{
		fakeError: errors.New("Fake error"),
	}

	cases := []struct {
		userClient       ClientInterface
		username         string
		objectID         string
		expectedUserType models.UserType
		expectedErr      error
	}{
		{
			userClient:       NewUserClient(config, azureClient),
			username:         "",
			objectID:         "00000000-0000-0000-0000-000000000000",
			expectedUserType: models.ServicePrincipalUserType,
			expectedErr:      nil,
		},
		{
			userClient:       NewUserClient(config, azureClient),
			username:         "username",
			objectID:         "00000000-0000-0000-0000-000000000000",
			expectedUserType: models.NormalUserType,
			expectedErr:      nil,
		},
		{
			userClient:       NewUserClient(config, azureClientError),
			username:         "username",
			objectID:         "00000000-0000-0000-0000-000000000000",
			expectedUserType: models.NormalUserType,
			expectedErr:      errors.New("Fake error"),
		},
	}

	for _, c := range cases {
		user, err := c.userClient.GetUser(ctx, c.username, c.objectID)

		if user.Type != c.expectedUserType && c.expectedErr == nil {
			t.Errorf("Expected user type (%s) was not returned: %s", c.expectedUserType, user.Type)
		}

		if err != nil && c.expectedErr == nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		if c.expectedErr != nil {
			if err.Error() != c.expectedErr.Error() {
				t.Errorf("Expected err to be %q but it was %q", c.expectedErr, err)
			}
		}
	}
}
