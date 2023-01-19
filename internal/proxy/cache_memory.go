package proxy

import (
	"context"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

type memoryCache struct {
	CacheClient *gocache.Cache
}

func newMemoryCache(expirationInterval time.Duration) (*memoryCache, error) {
	return &memoryCache{
		CacheClient: gocache.New(expirationInterval, 2*expirationInterval),
	}, nil
}

func (c *memoryCache) getUser(ctx context.Context, s string) (userModel, bool, error) {
	u, f := c.CacheClient.Get(s)
	if !f {
		return userModel{}, false, nil
	}
	return u.(userModel), true, nil
}

// SetUser ...
func (c *memoryCache) setUser(ctx context.Context, s string, u userModel) error {
	c.CacheClient.Set(s, u, 0)

	return nil
}

// GetGroup ...
func (c *memoryCache) getGroup(ctx context.Context, s string) (groupModel, bool, error) {
	g, f := c.CacheClient.Get(s)
	if !f {
		return groupModel{}, false, nil
	}
	return g.(groupModel), true, nil
}

// SetGroup ...
func (c *memoryCache) setGroup(ctx context.Context, s string, g groupModel) error {
	c.CacheClient.Set(s, g, 0)

	return nil
}
