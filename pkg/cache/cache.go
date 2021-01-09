package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// Cache ...
type Cache interface {
	GetUser(ctx context.Context, s string) (models.User, bool, error)
	SetUser(ctx context.Context, s string, u models.User) error
	GetGroup(ctx context.Context, s string) (models.Group, bool, error)
	SetGroup(ctx context.Context, s string, g models.Group) error
}

// NewCache ...
func NewCache(ctx context.Context, cacheEngine models.CacheEngine, config config.Config) (Cache, error) {
	switch cacheEngine {
	case models.RedisCacheEngine:
		return NewRedisCache(ctx, config.RedisURI, 5*time.Minute)
	case models.MemoryCacheEngine:
		return NewMemoryCache(5*time.Minute, 10*time.Minute)
	default:
		return nil, fmt.Errorf("Unknown cache engine: %s", cacheEngine)
	}
}
