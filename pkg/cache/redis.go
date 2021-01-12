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
	Expiration time.Duration
}

// NewRedisCache ...
func NewRedisCache(ctx context.Context, redisURL string, expiration time.Duration) (*RedisCache, error) {
	log := logr.FromContext(ctx)

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Error(err, "Unable to parse Redis URL", "redisURL", redisURL)
		return nil, err
	}

	redisClient := redis.NewClient(opt)
	err = redisClient.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return &RedisCache{
		Cache:      redisClient,
		Expiration: expiration,
	}, nil
}

// GetUser ...
func (c *RedisCache) GetUser(ctx context.Context, s string) (models.User, bool, error) {
	log := logr.FromContext(ctx)

	res := c.Cache.Get(ctx, s)
	err := res.Err()
	if err != nil {
		if err == redis.Nil {
			return models.User{}, false, nil
		}
		log.Error(err, "Unable to get user key from Redis cache", "keyName", s)
		return models.User{}, false, err
	}

	var u models.User

	err = res.Scan(&u)
	if err != nil { // no coverage in test
		log.Error(err, "Unable to unmarshal models.User from Redis cache value", "keyName", s)
		return models.User{}, false, err
	}

	return u, true, nil
}

// SetUser ...
func (c *RedisCache) SetUser(ctx context.Context, s string, u models.User) error {
	log := logr.FromContext(ctx)

	err := c.Cache.SetNX(ctx, s, u, c.Expiration).Err()
	if err != nil {
		log.Error(err, "Unable to cache user object to Redis", "keyName", s)
		return err
	}

	return nil
}

// GetGroup ...
func (c *RedisCache) GetGroup(ctx context.Context, s string) (models.Group, bool, error) {
	log := logr.FromContext(ctx)

	res := c.Cache.Get(ctx, s)
	err := res.Err()
	if err != nil {
		if err == redis.Nil {
			return models.Group{}, false, nil
		}

		log.Error(err, "Unable to get group key from Redis cache", "keyName", s)
		return models.Group{}, false, err
	}

	var g models.Group

	err = res.Scan(&g)
	if err != nil { // no coverage in test
		log.Error(err, "Unable to unmarshal models.Group from Redis cache value", "keyName", s)
		return models.Group{}, false, err
	}

	return g, true, nil
}

// SetGroup ...
func (c *RedisCache) SetGroup(ctx context.Context, s string, g models.Group) error {
	log := logr.FromContext(ctx)

	err := c.Cache.SetNX(ctx, s, g, c.Expiration).Err()
	if err != nil {
		log.Error(err, "Unable to cache group object to Redis", "keyName", s)
		return err
	}

	return nil
}
