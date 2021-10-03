package cache

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-logr/logr"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

func TestNewCache(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())

	cases := []struct {
		cacheEngine models.CacheEngine
		config      config.Config
		expectedErr error
	}{
		{
			cacheEngine: models.MemoryCacheEngine,
			config:      config.Config{},
			expectedErr: nil,
		},
		{
			cacheEngine: models.RedisCacheEngine,
			config: config.Config{
				RedisURI: redisURL,
			},
			expectedErr: nil,
		},
		{
			cacheEngine: models.RedisCacheEngine,
			config:      config.Config{},
			expectedErr: errors.New("redis: invalid URL scheme: "),
		},
		{
			cacheEngine: "",
			config:      config.Config{},
			expectedErr: errors.New("Unknown cache engine: "),
		},
		{
			cacheEngine: "DUMMY",
			config:      config.Config{},
			expectedErr: errors.New("Unknown cache engine: DUMMY"),
		},
	}

	for _, c := range cases {
		_, err := NewCache(ctx, c.cacheEngine, c.config)
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
