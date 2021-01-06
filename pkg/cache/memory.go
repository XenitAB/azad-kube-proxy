package cache

import (
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// MemoryCache ...
type MemoryCache struct {
	Cache             *gocache.Cache
	DefaultExpiration time.Duration
	CleanupInterval   time.Duration
}

// NewCache ...
func (c *MemoryCache) NewCache() {
	c.Cache = gocache.New(c.DefaultExpiration, c.CleanupInterval)
	return
}

// GetUser ...
func (c *MemoryCache) GetUser(s string) (models.User, bool, error) {
	u, f := c.Cache.Get(s)
	if !f {
		return models.User{}, false, nil
	}
	return u.(models.User), true, nil
}

// SetUser ...
func (c *MemoryCache) SetUser(s string, u models.User) error {
	c.Cache.Set(s, u, 0)

	return nil
}

// GetGroup ...
func (c *MemoryCache) GetGroup(s string) (models.Group, bool, error) {
	g, f := c.Cache.Get(s)
	if !f {
		return models.Group{}, false, nil
	}
	return g.(models.Group), true, nil
}

// SetGroup ...
func (c *MemoryCache) SetGroup(s string, g models.Group) error {
	c.Cache.Set(s, g, 0)

	return nil
}
