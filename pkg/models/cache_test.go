package models

import (
	"errors"
	"testing"
)

func TestGetCacheEngine(t *testing.T) {
	cases := []struct {
		cacheEngineString   string
		expectedCacheEngine CacheEngine
		expectedErr         error
	}{
		{
			cacheEngineString:   "MEMORY",
			expectedCacheEngine: MemoryCacheEngine,
			expectedErr:         nil,
		},
		{
			cacheEngineString:   "REDIS",
			expectedCacheEngine: RedisCacheEngine,
			expectedErr:         nil,
		},
		{
			cacheEngineString:   "",
			expectedCacheEngine: "",
			expectedErr:         errors.New("Unknown cache engine type ''. Supported engines are: MEMORY or REDIS"),
		},
		{
			cacheEngineString:   "DUMMY",
			expectedCacheEngine: "",
			expectedErr:         errors.New("Unknown cache engine type 'DUMMY'. Supported engines are: MEMORY or REDIS"),
		},
	}

	for _, c := range cases {
		resCacheEngine, err := GetCacheEngine(c.cacheEngineString)

		if resCacheEngine != c.expectedCacheEngine && c.expectedErr == nil {
			t.Errorf("Expected cacheEngine (%s) was not returned: %s", c.expectedCacheEngine, resCacheEngine)
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
