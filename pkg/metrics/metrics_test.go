package metrics

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

func TestNewMetricsClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.DiscardLogger{})
	cases := []struct {
		config              config.Config
		expectedErrContains string
	}{
		{
			config:              config.Config{},
			expectedErrContains: "Unexpected metrics:",
		},
		{
			config: config.Config{
				Metrics: models.NoneMetrics,
			},
			expectedErrContains: "",
		},
		{
			config: config.Config{
				Metrics: models.PrometheusMetrics,
			},
			expectedErrContains: "",
		},
	}

	for _, c := range cases {
		_, err := NewMetricsClient(ctx, c.config)
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
