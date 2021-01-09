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
	Expiration time.Duration
}

// NewRedisCache ...
func NewRedisCache(redisURL string, expiration time.Duration) (*RedisCache, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	return &RedisCache{
		Cache:      redis.NewClient(opt),
		Expiration: expiration,
	}, nil
}

// GetUser ...
func (c *RedisCache) GetUser(ctx context.Context, s string) (models.User, bool, error) {
	res := c.Cache.Get(ctx, s)
	if err := res.Err(); err != nil {
		if err == redis.Nil {
			return models.User{}, false, nil
		}
		return models.User{}, false, err
	}

	var u models.User

	err := res.Scan(&u)
	if err != nil {
		return models.User{}, false, err
	}

	return u, true, nil
}

// SetUser ...
func (c *RedisCache) SetUser(ctx context.Context, s string, u models.User) error {
	err := c.Cache.SetNX(ctx, s, u, c.Expiration).Err()
	if err != nil {
		return err
	}

	return nil
}

// GetGroup ...
func (c *RedisCache) GetGroup(ctx context.Context, s string) (models.Group, bool, error) {
	res := c.Cache.Get(ctx, s)
	if err := res.Err(); err != nil {
		if err == redis.Nil {
			return models.Group{}, false, nil
		}
		return models.Group{}, false, err
	}

	var g models.Group

	err := res.Scan(&g)
	if err != nil {
		return models.Group{}, false, err
	}

	return g, true, nil
}

// SetGroup ...
func (c *RedisCache) SetGroup(ctx context.Context, s string, g models.Group) error {
	err := c.Cache.SetNX(ctx, s, g, c.Expiration).Err()
	if err != nil {
		return err
	}

	return nil
}
