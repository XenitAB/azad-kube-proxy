package cache

import (
	"context"
	"fmt"
	"time"

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
func NewCache(cacheEngine models.CacheEngine) (Cache, error) {
	switch cacheEngine {
	case models.RedisCacheEngine:
		return NewRedisCache("127.0.0.1:6379", "", 0, 5*time.Minute), nil
	case models.MemoryCacheEngine:
		return NewMemoryCache(5*time.Minute, 10*time.Minute), nil
	default:
		return nil, fmt.Errorf("Unknown cache engine: %s", cacheEngine)
	}
}
