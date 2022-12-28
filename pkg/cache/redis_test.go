package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-logr/logr"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

var (
	testRedisTimeout = 5 * time.Minute
)

func TestNewRedisCache(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())

	_, err = NewRedisCache(ctx, redisURL, testRedisTimeout)
	require.NoError(t, err)

	_, err = NewRedisCache(ctx, "", testRedisTimeout)
	require.ErrorContains(t, err, "redis: invalid URL scheme: ")

	redisServer.Close()
	_, err = NewRedisCache(ctx, redisURL, testRedisTimeout)
	require.ErrorContains(t, err, "connect: connection refused")
}

func TestRedisGetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	require.NoError(t, err)

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	miniredisClient, err := testGetMiniredisClient(t, redisURL)
	require.NoError(t, err)
	defer redisServer.Close()

	cache, err := NewRedisCache(ctx, redisURL, testRedisTimeout)
	require.NoError(t, err)

	cases, _ := testGetRedisCases(t)

	for _, c := range cases {
		err := miniredisClient.SetNX(ctx, c.Key, c.User, testRedisTimeout).Err()
		require.NoError(t, err)
		cacheRes, found, err := cache.GetUser(ctx, c.Key)
		require.NoError(t, err)
		require.Equal(t, c.User, cacheRes)
		require.True(t, found)
	}

	// Not found
	_, found, _ := cache.GetUser(ctx, "does-not-exist")
	require.False(t, found)

	// Unmarshal error
	testFakeErrorUser := testFakeErrorUser{
		Username: false,
		t:        t,
	}
	err = miniredisClient.SetNX(ctx, "fake-error-user", testFakeErrorUser, testRedisTimeout).Err()
	require.NoError(t, err)

	_, _, err = cache.GetUser(ctx, "fake-error-user")
	require.ErrorContains(t, err, "json: cannot unmarshal")

	// Connection error
	redisServer.Close()
	_, _, err = cache.GetUser(ctx, "no-redis-server")
	require.ErrorContains(t, err, "connect: connection refused")
}

func TestRedisSetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	miniredisClient, err := testGetMiniredisClient(t, redisURL)
	require.NoError(t, err)

	cache, err := NewRedisCache(ctx, redisURL, testRedisTimeout)
	require.NoError(t, err)

	cases, _ := testGetRedisCases(t)

	for _, c := range cases {
		err := cache.SetUser(ctx, c.Key, c.User)
		require.NoError(t, err)

		res := miniredisClient.Get(ctx, c.Key)
		require.NoError(t, res.Err())

		var u models.User
		err = res.Scan(&u)
		require.NoError(t, err)
		require.Equal(t, c.User, u)
	}

	redisServer.Close()
	err = cache.SetUser(ctx, "no-redis-server", models.User{})
	require.ErrorContains(t, err, "connect: connection refused")
}

func TestRedisGetGroup(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	miniredisClient, err := testGetMiniredisClient(t, redisURL)
	require.NoError(t, err)

	cache, err := NewRedisCache(ctx, redisURL, testRedisTimeout)
	require.NoError(t, err)

	_, cases := testGetRedisCases(t)

	for _, c := range cases {
		err := miniredisClient.SetNX(ctx, c.Key, c.Group, testRedisTimeout).Err()
		require.NoError(t, err)
		cacheRes, found, err := cache.GetGroup(ctx, c.Key)
		require.NoError(t, err)
		require.Equal(t, c.Group, cacheRes)
		require.True(t, found)
	}

	// Not found
	_, found, _ := cache.GetGroup(ctx, "does-not-exist")
	require.False(t, found)

	// Unmarshal error
	testFakeErrorGroup := testFakeErrorGroup{
		Name: false,
		t:    t,
	}
	err = miniredisClient.SetNX(ctx, "fake-error-group", testFakeErrorGroup, testRedisTimeout).Err()
	require.NoError(t, err)

	_, _, err = cache.GetGroup(ctx, "fake-error-group")
	require.ErrorContains(t, err, "json: cannot unmarshal")

	// Connection error
	redisServer.Close()
	_, _, err = cache.GetGroup(ctx, "no-redis-server")
	require.ErrorContains(t, err, "connect: connection refused")
}

func TestRedisSetGroup(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	require.NoError(t, err)
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	miniredisClient, err := testGetMiniredisClient(t, redisURL)
	require.NoError(t, err)

	cache, err := NewRedisCache(ctx, redisURL, testRedisTimeout)
	require.NoError(t, err)

	_, cases := testGetRedisCases(t)

	for _, c := range cases {
		err := cache.SetGroup(ctx, c.Key, c.Group)
		require.NoError(t, err)

		res := miniredisClient.Get(ctx, c.Key)
		require.NoError(t, res.Err())

		var g models.Group

		err = res.Scan(&g)
		require.NoError(t, err)
		require.Equal(t, c.Group, g)
	}

	redisServer.Close()
	err = cache.SetGroup(ctx, "no-redis-server", models.Group{})
	require.ErrorContains(t, err, "connect: connection refused")
}

func testGetMiniredisClient(t *testing.T, redisURL string) (*redis.Client, error) {
	t.Helper()

	opt, err := redis.ParseURL(redisURL)
	require.NoError(t, err)
	return redis.NewClient(opt), err
}

type testFakeErrorUser struct {
	Username bool
	t        *testing.T
}

func (i testFakeErrorUser) MarshalBinary() ([]byte, error) {
	i.t.Helper()

	return json.Marshal(i)
}

type testFakeErrorGroup struct {
	Name bool
	t    *testing.T
}

func (i testFakeErrorGroup) MarshalBinary() ([]byte, error) {
	i.t.Helper()

	return json.Marshal(i)
}

type testRedisUserCase struct {
	User models.User
	Key  string
}

type testRedisGroupCase struct {
	Group models.Group
	Key   string
}

func testGetRedisCases(t *testing.T) ([]testRedisUserCase, []testRedisGroupCase) {
	t.Helper()

	userCases := []testRedisUserCase{
		{
			User: models.User{
				Username: "user1",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []models.Group{
					{
						Name: "group1",
					},
				},
				Type: models.NormalUserType,
			},
			Key: "tokenHash1",
		},
		{
			User: models.User{
				Username: "user2",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []models.Group{
					{
						Name: "group2",
					},
				},
				Type: models.NormalUserType,
			},
			Key: "tokenHash2",
		},
		{
			User: models.User{
				Username: "00000000-0000-0000-0000-000000000000",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []models.Group{
					{
						Name: "group1",
					},
					{
						Name: "group2",
					},
					{
						Name: "group3",
					},
				},
				Type: models.ServicePrincipalUserType,
			},
			Key: "tokenHash3",
		},
	}

	groupCases := []testRedisGroupCase{
		{
			Group: models.Group{Name: "group1"},
			Key:   "00000000-0000-0000-0000-000000000000",
		},
		{
			Group: models.Group{Name: "group2"},
			Key:   "00000000-0000-0000-0000-000000000001",
		},
		{
			Group: models.Group{Name: "group3"},
			Key:   "00000000-0000-0000-0000-000000000002",
		},
	}

	return userCases, groupCases
}
