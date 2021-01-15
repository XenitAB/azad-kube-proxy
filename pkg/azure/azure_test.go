package azure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-logr/logr"
	logrTesting "github.com/go-logr/logr/testing"
	"github.com/xenitab/azad-kube-proxy/pkg/cache"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

func TestNewAzureClient(t *testing.T) {
	clientID := getEnvOrSkip(t, "CLIENT_ID")
	clientSecret := getEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	userObjectID := getEnvOrSkip(t, "TEST_USER_OBJECT_ID")
	spObjectID := getEnvOrSkip(t, "TEST_USER_SP_OBJECT_ID")
	graphFilter := ""
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	cache, err := cache.NewCache(ctx, models.MemoryCacheEngine, config.Config{})
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	azureClient, err := NewAzureClient(ctx, clientID, clientSecret, tenantID, graphFilter, cache)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, err = azureClient.GetUserGroups(ctx, userObjectID, models.NormalUserType)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, err = azureClient.GetUserGroups(ctx, "00000000-0000-0000-0000-000000000000", models.NormalUserType)
	if err == nil {
		t.Errorf("Expected err not to be nil but it was")
	}

	_, err = azureClient.GetUserGroups(ctx, spObjectID, models.ServicePrincipalUserType)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, err = azureClient.GetUserGroups(ctx, "00000000-0000-0000-0000-000000000000", models.ServicePrincipalUserType)
	if err == nil {
		t.Errorf("Expected err not to be nil but it was")
	}

	groupSyncTicker, groupSyncChan, err := azureClient.StartSyncGroups(ctx, 5*time.Minute)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	var stopGroupSync func() = func() {
		groupSyncTicker.Stop()
		groupSyncChan <- true
		return
	}
	defer stopGroupSync()
}

func getEnvOrSkip(t *testing.T, envVar string) string {
	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s environment variable is empty, skipping.", envVar)
	}

	return v
}
