package user

import (
	"context"
	"testing"
	"time"

	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

type FakeAzureClient struct{}

// GetUserGroups ...
func (client *FakeAzureClient) GetUserGroups(ctx context.Context, objectID string, userType models.UserType) ([]models.Group, error) {
	return nil, nil
}

// StartSyncGroups initiates a ticker that will sync Azure AD Groups
func (client *FakeAzureClient) StartSyncGroups(ctx context.Context, syncInterval time.Duration) (*time.Ticker, chan bool, error) {
	return nil, nil, nil
}

func TestGetUser(t *testing.T) {
	config := config.Config{}
	azureClient := &FakeAzureClient{}
	userClient := NewUserClient(config, azureClient)

	ctx := context.Background()
	user, userErr := userClient.GetUser(ctx, "username", "00000000-0000-0000-0000-000000000000")

	if user.Type != models.NormalUserType {
		t.Errorf("Normal user: Expected user type was not returned: %s", user.Type)
	}
	if userErr != nil {
		t.Errorf("Normal user returned error: %s", userErr)
	}

	spUser, spErr := userClient.GetUser(ctx, "", "00000000-0000-0000-0000-000000000000")
	if spUser.Type != models.ServicePrincipalUserType {
		t.Errorf("Service principal: Expected user type was not returned: %s", spUser.Type)
	}
	if spErr != nil {
		t.Errorf("Service principal returned error: %s", spErr)
	}
}
