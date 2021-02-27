package models

import (
	"errors"
	"testing"
)

func TestGetMetrics(t *testing.T) {
	cases := []struct {
		metricsString   string
		expectedMetrics Metrics
		expectedErr     error
	}{
		{
			metricsString:   "NONE",
			expectedMetrics: NoneMetrics,
			expectedErr:     nil,
		},
		{
			metricsString:   "PROMETHEUS",
			expectedMetrics: PrometheusMetrics,
			expectedErr:     nil,
		},
		{
			metricsString:   "",
			expectedMetrics: "",
			expectedErr:     errors.New("Unknown metrics ''. Supported engines are: NONE or PROMETHEUS"),
		},
		{
			metricsString:   "DUMMY",
			expectedMetrics: "",
			expectedErr:     errors.New("Unknown metrics 'DUMMY'. Supported engines are: NONE or PROMETHEUS"),
		},
	}

	for _, c := range cases {
		resMetrics, err := GetMetrics(c.metricsString)

		if resMetrics != c.expectedMetrics && c.expectedErr == nil {
			t.Errorf("Expected cacheEngine (%s) was not returned: %s", c.expectedMetrics, resMetrics)
		}

		if err != nil && c.expectedErr == nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		if c.expectedErr != nil {
			if err.Error() != c.expectedErr.Error() {
				t.Errorf("Expected err to be %q but it was %q", c.expectedErr, err)
			}
		}
	}
}
