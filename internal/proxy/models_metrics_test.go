package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetMetrics(t *testing.T) {
	cases := []struct {
		metricsString       string
		expectedMetrics     metricsModel
		expectedErrContains string
	}{
		{
			metricsString:       "NONE",
			expectedMetrics:     noneMetrics,
			expectedErrContains: "",
		},
		{
			metricsString:       "PROMETHEUS",
			expectedMetrics:     prometheusMetrics,
			expectedErrContains: "",
		},
		{
			metricsString:       "",
			expectedMetrics:     "",
			expectedErrContains: "Unknown metrics ''. Supported engines are: NONE or PROMETHEUS",
		},
		{
			metricsString:       "DUMMY",
			expectedMetrics:     "",
			expectedErrContains: "Unknown metrics 'DUMMY'. Supported engines are: NONE or PROMETHEUS",
		},
	}

	for _, c := range cases {
		resMetrics, err := getMetrics(c.metricsString)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.expectedMetrics, resMetrics)
	}
}
