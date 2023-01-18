package proxy

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsClient(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cases := []struct {
		config              *Config
		expectedErrContains string
	}{
		{
			config:              &Config{},
			expectedErrContains: "Unknown metrics",
		},
		{
			config: &Config{
				Metrics: "NONE",
			},
			expectedErrContains: "",
		},
		{
			config: &Config{
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
