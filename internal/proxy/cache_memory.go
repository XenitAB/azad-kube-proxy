package proxy

import (
	"context"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/xenitab/azad-kube-proxy/internal/models"
)

type memoryCache struct {
	CacheClient *gocache.Cache
}

func newMemoryCache(expirationInterval time.Duration) (*memoryCache, error) {
	return &memoryCache{
		CacheClient: gocache.New(expirationInterval, 2*expirationInterval),
	}, nil
}

// GetUser ...
func (c *memoryCache) GetUser(ctx context.Context, s string) (models.User, bool, error) {
	u, f := c.CacheClient.Get(s)
	if !f {
		return models.User{}, false, nil
	}
	return u.(models.User), true, nil
}

// SetUser ...
func (c *memoryCache) SetUser(ctx context.Context, s string, u models.User) error {
	c.CacheClient.Set(s, u, 0)

	return nil
}

// GetGroup ...
func (c *memoryCache) GetGroup(ctx context.Context, s string) (models.Group, bool, error) {
	g, f := c.CacheClient.Get(s)
	if !f {
		return models.Group{}, false, nil
	}
	return g.(models.Group), true, nil
}

// SetGroup ...
func (c *memoryCache) SetGroup(ctx context.Context, s string, g models.Group) error {
	c.CacheClient.Set(s, g, 0)

	return nil
}
