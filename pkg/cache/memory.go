package cache

import (
	"context"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// MemoryCache ...
type MemoryCache struct {
	CacheClient *gocache.Cache
}

// NewMemoryCache ...
func NewMemoryCache(defaultExpiration, cleanupInterval time.Duration) (*MemoryCache, error) {
	return &MemoryCache{
		CacheClient: gocache.New(defaultExpiration, cleanupInterval),
	}, nil
}

// GetUser ...
func (c *MemoryCache) GetUser(ctx context.Context, s string) (models.User, bool, error) {
	u, f := c.CacheClient.Get(s)
	if !f {
		return models.User{}, false, nil
	}
	return u.(models.User), true, nil
}

// SetUser ...
func (c *MemoryCache) SetUser(ctx context.Context, s string, u models.User) error {
	c.CacheClient.Set(s, u, 0)

	return nil
}

// GetGroup ...
func (c *MemoryCache) GetGroup(ctx context.Context, s string) (models.Group, bool, error) {
	g, f := c.CacheClient.Get(s)
	if !f {
		return models.Group{}, false, nil
	}
	return g.(models.Group), true, nil
}

// SetGroup ...
func (c *MemoryCache) SetGroup(ctx context.Context, s string, g models.Group) error {
	c.CacheClient.Set(s, g, 0)

	return nil
}
