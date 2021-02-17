package models

import (
	"errors"
	"testing"
)

func TestGetDashboard(t *testing.T) {
	cases := []struct {
		dashboardString   string
		expectedDashboard Dashboard
		expectedErr       error
	}{
		{
			dashboardString:   "NONE",
			expectedDashboard: NoneDashboard,
			expectedErr:       nil,
		},
		{
			dashboardString:   "K8DASH",
			expectedDashboard: K8dashDashboard,
			expectedErr:       nil,
		},
		{
			dashboardString:   "",
			expectedDashboard: "",
			expectedErr:       errors.New("Unknown dashboard ''. Supported engines are: NONE or K8DASH"),
		},
		{
			dashboardString:   "DUMMY",
			expectedDashboard: "",
			expectedErr:       errors.New("Unknown dashboard 'DUMMY'. Supported engines are: NONE or K8DASH"),
		},
	}

	for _, c := range cases {
		resDashboard, err := GetDashboard(c.dashboardString)

		if resDashboard != c.expectedDashboard && c.expectedErr == nil {
			t.Errorf("Expected cacheEngine (%s) was not returned: %s", c.expectedDashboard, resDashboard)
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
