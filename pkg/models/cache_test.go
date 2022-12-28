package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCacheEngine(t *testing.T) {
	cases := []struct {
		cacheEngineString   string
		expectedCacheEngine CacheEngine
		expectedErrContains string
	}{
		{
			cacheEngineString:   "MEMORY",
			expectedCacheEngine: MemoryCacheEngine,
			expectedErrContains: "",
		},
		{
			cacheEngineString:   "REDIS",
			expectedCacheEngine: RedisCacheEngine,
			expectedErrContains: "",
		},
		{
			cacheEngineString:   "",
			expectedCacheEngine: "",
			expectedErrContains: "Unknown cache engine type ''. Supported engines are: MEMORY or REDIS",
		},
		{
			cacheEngineString:   "DUMMY",
			expectedCacheEngine: "",
			expectedErrContains: "Unknown cache engine type 'DUMMY'. Supported engines are: MEMORY or REDIS",
		},
	}

	for _, c := range cases {
		resCacheEngine, err := GetCacheEngine(c.cacheEngineString)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}

		require.NoError(t, err)
		require.Equal(t, c.expectedCacheEngine, resCacheEngine)
	}
}
