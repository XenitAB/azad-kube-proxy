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
		config              *config
		expectedErrContains string
	}{
		{
			config:              &config{},
			expectedErrContains: "Unknown metrics",
		},
		{
			config: &config{
				Metrics: "NONE",
			},
			expectedErrContains: "",
		},
		{
			config: &config{
				Metrics: "PROMETHEUS",
			},
			expectedErrContains: "",
		},
	}

	for _, c := range cases {
		_, err := newMetricsClient(ctx, c.config)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
	}
}
