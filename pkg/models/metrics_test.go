package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetMetrics(t *testing.T) {
	cases := []struct {
		metricsString       string
		expectedMetrics     Metrics
		expectedErrContains string
	}{
		{
			metricsString:       "NONE",
			expectedMetrics:     NoneMetrics,
			expectedErrContains: "",
		},
		{
			metricsString:       "PROMETHEUS",
			expectedMetrics:     PrometheusMetrics,
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
		resMetrics, err := GetMetrics(c.metricsString)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.expectedMetrics, resMetrics)
	}
}
