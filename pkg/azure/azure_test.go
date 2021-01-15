package azure

import (
	"context"
	"os"
	"strings"
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
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	memCache, err := cache.NewCache(ctx, models.MemoryCacheEngine, config.Config{})
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	cases := []struct {
		clientID            string
		clientSecret        string
		tenantID            string
		graphFilter         string
		cache               cache.Cache
		expectedErrContains string
	}{
		{
			clientID:            clientID,
			clientSecret:        clientSecret,
			tenantID:            tenantID,
			graphFilter:         "",
			cache:               memCache,
			expectedErrContains: "",
		},
		{
			clientID:            clientID,
			clientSecret:        clientSecret,
			tenantID:            tenantID,
			graphFilter:         "prefix",
			cache:               memCache,
			expectedErrContains: "",
		},
		{
			clientID:            clientID,
			clientSecret:        clientSecret,
			tenantID:            "",
			graphFilter:         "",
			cache:               memCache,
			expectedErrContains: "Client Secret Credential: Invalid tenantID provided.",
		},
		{
			clientID:            "",
			clientSecret:        "",
			tenantID:            tenantID,
			graphFilter:         "",
			cache:               memCache,
			expectedErrContains: "AADSTS7000216",
		},
	}

	for _, c := range cases {
		_, err := NewAzureClient(ctx, c.clientID, c.clientSecret, c.tenantID, c.graphFilter, c.cache)
		if err != nil && len(c.expectedErrContains) == 0 {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		if len(c.expectedErrContains) > 0 {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain %q but it was %q", c.expectedErrContains, err)
			}
		}
	}
}

func TestGetUserGroups(t *testing.T) {
	clientID := getEnvOrSkip(t, "CLIENT_ID")
	clientSecret := getEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	userObjectID := getEnvOrSkip(t, "TEST_USER_OBJECT_ID")
	spObjectID := getEnvOrSkip(t, "TEST_USER_SP_OBJECT_ID")
	graphFilter := ""
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	memCache, err := cache.NewCache(ctx, models.MemoryCacheEngine, config.Config{})
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	azureClient, err := NewAzureClient(ctx, clientID, clientSecret, tenantID, graphFilter, memCache)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

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
			expectedErrContains: "Failure responding to request: StatusCode=405",
		},
		{
			objectID:            "",
			userType:            models.ServicePrincipalUserType,
			expectedErrContains: "Status code not 200 OK: 400",
		},
	}

	for _, c := range cases {
		_, err := azureClient.GetUserGroups(ctx, c.objectID, c.userType)
		if err != nil && len(c.expectedErrContains) == 0 {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		if len(c.expectedErrContains) > 0 {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain %q but it was %q", c.expectedErrContains, err)
			}
		}
	}
}

func TestStartSyncGroups(t *testing.T) {
	clientID := getEnvOrSkip(t, "CLIENT_ID")
	clientSecret := getEnvOrSkip(t, "CLIENT_SECRET")
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	graphFilter := ""
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	memCache, err := cache.NewCache(ctx, models.MemoryCacheEngine, config.Config{})
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	azureClient, err := NewAzureClient(ctx, clientID, clientSecret, tenantID, graphFilter, memCache)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	groupSyncTicker, groupSyncChan, err := azureClient.StartSyncGroups(ctx, 1*time.Second)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	time.Sleep(2 * time.Second)
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
