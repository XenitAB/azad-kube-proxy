package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// RedisCache ...
type RedisCache struct {
	Cache      *redis.Client
	Address    string
	Context    context.Context
	Expiration time.Duration
}

// NewCache ...
func (c *RedisCache) NewCache() {
	c.Cache = redis.NewClient(&redis.Options{
		Addr: c.Address,
	})

	return
}

// GetUser ...
func (c *RedisCache) GetUser(s string) (models.User, bool) {
	res := c.Cache.Get(c.Context, s)

	var u models.User

	err := res.Scan(&u)
	if err != nil {
		return models.User{}, false
	}

	return u, true
}

// SetUser ...
func (c *RedisCache) SetUser(s string, u models.User) error {
	err := c.Cache.Set(c.Context, s, u, c.Expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

// GetGroup ...
func (c *RedisCache) GetGroup(s string) (models.Group, bool) {
	res := c.Cache.Get(c.Context, s)

	var g models.Group

	err := res.Scan(&g)
	if err != nil {
		return models.Group{}, false
	}

	return g, true
}

// SetGroup ...
func (c *RedisCache) SetGroup(s string, g models.Group) error {
	err := c.Cache.Set(c.Context, s, g, c.Expiration).Err()
	if err != nil {
		return err
	}

	return nil
}
