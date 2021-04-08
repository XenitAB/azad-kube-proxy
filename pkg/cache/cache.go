package cache

import (
	"context"
	"fmt"

	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// ClientInterface ...
type ClientInterface interface {
	GetUser(ctx context.Context, s string) (models.User, bool, error)
	SetUser(ctx context.Context, s string, u models.User) error
	GetGroup(ctx context.Context, s string) (models.Group, bool, error)
	SetGroup(ctx context.Context, s string, g models.Group) error
}

// NewCache ...
func NewCache(ctx context.Context, cacheEngine models.CacheEngine, config config.Config) (ClientInterface, error) {
	ttl := 4 * config.GroupSyncInterval

	switch cacheEngine {
	case models.RedisCacheEngine:
		return NewRedisCache(ctx, config.RedisURI, ttl)
	case models.MemoryCacheEngine:
		return NewMemoryCache(ttl, 2*ttl)
	default:
		return nil, fmt.Errorf("Unknown cache engine: %s", cacheEngine)
	}
}
