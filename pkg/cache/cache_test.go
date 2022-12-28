package cache

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

func TestNewCache(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())

	cases := []struct {
		cacheEngine         models.CacheEngine
		config              config.Config
		expectedErrContains string
	}{
		{
			cacheEngine:         models.MemoryCacheEngine,
			config:              config.Config{},
			expectedErrContains: "",
		},
		{
			cacheEngine: models.RedisCacheEngine,
			config: config.Config{
				RedisURI: redisURL,
			},
			expectedErrContains: "",
		},
		{
			cacheEngine:         models.RedisCacheEngine,
			config:              config.Config{},
			expectedErrContains: "redis: invalid URL scheme: ",
		},
		{
			cacheEngine:         "",
			config:              config.Config{},
			expectedErrContains: "Unknown cache engine: ",
		},
		{
			cacheEngine:         "DUMMY",
			config:              config.Config{},
			expectedErrContains: "Unknown cache engine: DUMMY",
		},
	}

	for _, c := range cases {
		_, err := NewCache(ctx, c.cacheEngine, c.config)
		if c.expectedErrContains != "" {
			require.ErrorContains(t, err, c.expectedErrContains)
			continue
		}
		require.NoError(t, err)
	}
}
