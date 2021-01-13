package config

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	logrTesting "github.com/go-logr/logr/testing"
	"github.com/google/go-cmp/cmp"
)

// ./azad-kube-proxy --test abc --hejsan 123
// [./azad-kube-proxy --test abc --hejsan 123]
func TestGetConfig(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})

	cases := []struct {
		osArgs         []string
		expectedConfig Config
		expectedErr    error
	}{
		{
			osArgs:         []string{""},
			expectedConfig: Config{},
			expectedErr:    nil,
		},
	}

	for _, c := range cases {
		_, _ = GetConfig(ctx, c.osArgs)
		if !cmp.Equal(c.expectedConfig, Config{}) {
			t.Log("Config is empty")
			// if config[c.expectedKey] != c.expectedUserType && c.expectedErr == nil {
			// 	t.Errorf("Expected user type (%s) was not returned: %s", c.expectedUserType, user.Type)
			// }
		}

		// if err != nil && c.expectedErr == nil {
		// 	t.Errorf("Expected err to be nil but it was %q", err)
		// }

		// if c.expectedErr != nil {
		// 	if err.Error() != c.expectedErr.Error() {
		// 		t.Errorf("Expected err to be %q but it was %q", c.expectedErr, err)
		// 	}
		// }
	}
}
