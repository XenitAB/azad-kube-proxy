package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserAgentToKubectlVersion(t *testing.T) {
	cases := []struct {
		userAgent       string
		expectedVersion string
	}{
		{
			userAgent:       "kubectl/v1.22.2 (linux/amd64) kubernetes/8b5a191",
			expectedVersion: "v1.22.2",
		},
		{
			userAgent:       "foobar",
			expectedVersion: "unknown",
		},
	}

	for _, c := range cases {
		result := userAgentToKubectlVersion(c.userAgent)
		require.Equal(t, c.expectedVersion, result)
	}
}
