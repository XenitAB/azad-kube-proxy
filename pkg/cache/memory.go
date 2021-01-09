package cache

import (
	"context"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// MemoryCache ...
type MemoryCache struct {
	Cache *gocache.Cache
}

// NewMemoryCache ...
func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) *MemoryCache {
	return &MemoryCache{
		Cache: gocache.New(defaultExpiration, cleanupInterval),
	}
}

// GetUser ...
func (c *MemoryCache) GetUser(ctx context.Context, s string) (models.User, bool, error) {
	u, f := c.Cache.Get(s)
	if !f {
		return models.User{}, false, nil
	}
	return u.(models.User), true, nil
}

// SetUser ...
func (c *MemoryCache) SetUser(ctx context.Context, s string, u models.User) error {
	c.Cache.Set(s, u, 0)

	return nil
}

// GetGroup ...
func (c *MemoryCache) GetGroup(ctx context.Context, s string) (models.Group, bool, error) {
	g, f := c.Cache.Get(s)
	if !f {
		return models.Group{}, false, nil
	}
	return g.(models.Group), true, nil
}

// SetGroup ...
func (c *MemoryCache) SetGroup(ctx context.Context, s string, g models.Group) error {
	c.Cache.Set(s, g, 0)

	return nil
}
