package metrics

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

func TestNewMetricsClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cases := []struct {
		config              *config.Config
		expectedErrContains string
	}{
		{
			config:              &config.Config{},
			expectedErrContains: "Unknown metrics",
		},
		{
			config: &config.Config{
				Metrics: "NONE",
			},
			expectedErrContains: "",
		},
		{
			config: &config.Config{
				Metrics: "PROMETHEUS",
			},
			expectedErrContains: "",
		},
	}

	for _, c := range cases {
		_, err := NewMetricsClient(ctx, c.config)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
	}
}
