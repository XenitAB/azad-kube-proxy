package cache

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-redis/redis/v8"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// RedisCache ...
type RedisCache struct {
	Cache      *redis.Client
	Address    string
	Password   string
	Database   int
	Context    context.Context
	Expiration time.Duration
}

// NewCache ...
func (c *RedisCache) NewCache() {
	c.Cache = redis.NewClient(&redis.Options{
		Addr:     c.Address,
		Password: c.Password,
		DB:       c.Database,
	})

	return
}

// GetUser ...
func (c *RedisCache) GetUser(s string) (models.User, bool) {
	log := logr.FromContext(c.Context)

	res := c.Cache.Get(c.Context, s)
	if err := res.Err(); err != nil {
		if err == redis.Nil {
			return models.User{}, false
		}

		log.Error(err, "Failed to get key from redis cache", "key", s)
		return models.User{}, false
	}

	var u models.User

	err := res.Scan(&u)
	if err != nil {
		return models.User{}, false
	}

	return u, true
}

// SetUser ...
func (c *RedisCache) SetUser(s string, u models.User) error {
	err := c.Cache.SetNX(c.Context, s, u, c.Expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

// GetGroup ...
func (c *RedisCache) GetGroup(s string) (models.Group, bool) {
	log := logr.FromContext(c.Context)

	res := c.Cache.Get(c.Context, s)
	if err := res.Err(); err != nil {
		if err == redis.Nil {
			return models.Group{}, false
		}

		log.Error(err, "Failed to get key from redis cache", "key", s)
		return models.Group{}, false
	}

	var g models.Group

	err := res.Scan(&g)
	if err != nil {
		return models.Group{}, false
	}

	return g, true
}

// SetGroup ...
func (c *RedisCache) SetGroup(s string, g models.Group) error {
	err := c.Cache.SetNX(c.Context, s, g, c.Expiration).Err()
	if err != nil {
		return err
	}

	return nil
}
