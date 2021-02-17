package dashboard

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

func TestNewDashboardClient(t *testing.T) {
	tenantID := getEnvOrSkip(t, "TENANT_ID")
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	cases := []struct {
		config              config.Config
		expectedErrContains string
	}{
		{
			config:              config.Config{},
			expectedErrContains: "Unexpected dashboard:",
		},
		{
			config: config.Config{
				Dashboard: models.NoneDashboard,
			},
			expectedErrContains: "",
		},
		{
			config: config.Config{
				Dashboard: models.K8dashDashboard,
				TenantID:  tenantID,
			},
			expectedErrContains: "",
		},
		{
			config: config.Config{
				Dashboard: models.K8dashDashboard,
			},
			expectedErrContains: "400 Bad Request",
		},
	}

	for _, c := range cases {
		_, err := NewDashboardClient(ctx, c.config)
		if err != nil && c.expectedErrContains == "" {
			t.Errorf("Expected err to be nil: %q", err)
		}

		if err == nil && c.expectedErrContains != "" {
			t.Errorf("Expected err to contain '%s' but was nil", c.expectedErrContains)
		}

		if err != nil && c.expectedErrContains != "" {
			if !strings.Contains(err.Error(), c.expectedErrContains) {
				t.Errorf("Expected err to contain '%s' but was: %q", c.expectedErrContains, err)
			}
		}
	}
}

func getEnvOrSkip(t *testing.T, envVar string) string {
	v := os.Getenv(envVar)
	if v == "" {
		t.Skipf("%s environment variable is empty, skipping.", envVar)
	}

	return v
}
