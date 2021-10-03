package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-logr/logr"
	"github.com/go-redis/redis/v8"
	"github.com/google/go-cmp/cmp"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

var (
	redisTimeout = 5 * time.Minute
)

type fakeErrorUser struct {
	Username bool
}

func (i fakeErrorUser) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

type fakeErrorGroup struct {
	Name bool
}

func (i fakeErrorGroup) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

func TestNewRedisCache(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())

	_, err = NewRedisCache(ctx, redisURL, redisTimeout)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, err = NewRedisCache(ctx, "", redisTimeout)
	if err.Error() != "redis: invalid URL scheme: " {
		t.Errorf("Expected err to contain 'redis: invalid URL scheme: ' but was: %q", err)
	}

	redisServer.Close()
	_, err = NewRedisCache(ctx, redisURL, redisTimeout)
	if !strings.Contains(err.Error(), "connect: connection refused") {
		t.Errorf("Expected err to contain 'connect: connection refused' but was: %q", err)
	}
}

func TestRedisGetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	miniredisClient, err := getMiniredisClient(redisURL)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	defer redisServer.Close()

	cache, err := NewRedisCache(ctx, redisURL, redisTimeout)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	cases, _ := getRedisCases()

	for _, c := range cases {
		err := miniredisClient.SetNX(ctx, c.Key, c.User, redisTimeout).Err()
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}
		cacheRes, found, err := cache.GetUser(ctx, c.Key)
		if !cmp.Equal(c.User, cacheRes) {
			t.Errorf("Expected response was not returned.\nExpected: %s\nActual:   %s", c.User, cacheRes)
		}
		if err != nil {
			t.Errorf("Did not expect error: %q", err)
		}
		if !found {
			t.Errorf("Expected cached user to be found")
		}
	}

	// Not found
	_, found, _ := cache.GetUser(ctx, "does-not-exist")
	if found {
		t.Errorf("Expected cached user not to be found")
	}

	// Unmarshal error
	fakeErrorUser := fakeErrorUser{
		Username: false,
	}
	err = miniredisClient.SetNX(ctx, "fake-error-user", fakeErrorUser, redisTimeout).Err()
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, _, err = cache.GetUser(ctx, "fake-error-user")
	if !strings.Contains(err.Error(), "json: cannot unmarshal") {
		t.Errorf("Expected error to contain 'json: cannot unmarshal' but it was: %q", err)
	}

	// Connection error
	redisServer.Close()
	_, _, err = cache.GetUser(ctx, "no-redis-server")
	if !strings.Contains(err.Error(), "connect: connection refused") {
		t.Errorf("Expected error to contain 'connect: connection refused' but it was: %q", err)
	}
}

func TestRedisSetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	miniredisClient, err := getMiniredisClient(redisURL)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	cache, err := NewRedisCache(ctx, redisURL, redisTimeout)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	cases, _ := getRedisCases()

	for _, c := range cases {
		err := cache.SetUser(ctx, c.Key, c.User)
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		res := miniredisClient.Get(ctx, c.Key)
		found := true
		err = res.Err()
		if err != nil {
			if err == redis.Nil {
				found = false
			} else {
				t.Errorf("Expected err to be nil but it was %q", err)
			}
		}

		var u models.User

		err = res.Scan(&u)
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		if !cmp.Equal(c.User, u) {
			t.Errorf("Expected response was not returned.\nExpected: %s\nActual:   %s", c.User, u)
		}
		if err != nil {
			t.Errorf("Did not expect error: %q", err)
		}
		if !found {
			t.Errorf("Expected cached user to be found")
		}
	}

	redisServer.Close()
	err = cache.SetUser(ctx, "no-redis-server", models.User{})
	if !strings.Contains(err.Error(), "connect: connection refused") {
		t.Errorf("Expected error to contain 'connect: connection refused' but it was: %q", err)
	}
}

func TestRedisGetGroup(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	miniredisClient, err := getMiniredisClient(redisURL)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	cache, err := NewRedisCache(ctx, redisURL, redisTimeout)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, cases := getRedisCases()

	for _, c := range cases {
		err := miniredisClient.SetNX(ctx, c.Key, c.Group, redisTimeout).Err()
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}
		cacheRes, found, err := cache.GetGroup(ctx, c.Key)
		if !cmp.Equal(c.Group, cacheRes) {
			t.Errorf("Expected response was not returned.\nExpected: %s\nActual:   %s", c.Group, cacheRes)
		}
		if err != nil {
			t.Errorf("Did not expect error: %q", err)
		}
		if !found {
			t.Errorf("Expected cached group to be found")
		}
	}

	// Not found
	_, found, _ := cache.GetGroup(ctx, "does-not-exist")
	if found {
		t.Errorf("Expected cached group not to be found")
	}

	// Unmarshal error
	fakeErrorGroup := fakeErrorGroup{
		Name: false,
	}
	err = miniredisClient.SetNX(ctx, "fake-error-group", fakeErrorGroup, redisTimeout).Err()
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, _, err = cache.GetGroup(ctx, "fake-error-group")
	if !strings.Contains(err.Error(), "json: cannot unmarshal") {
		t.Errorf("Expected error to contain 'json: cannot unmarshal' but it was: %q", err)
	}

	// Connection error
	redisServer.Close()
	_, _, err = cache.GetGroup(ctx, "no-redis-server")
	if !strings.Contains(err.Error(), "connect: connection refused") {
		t.Errorf("Expected error to contain 'connect: connection refused' but it was: %q", err)
	}
}

func TestRedisSetGroup(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	redisServer, err := miniredis.Run()
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}
	defer redisServer.Close()

	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
	miniredisClient, err := getMiniredisClient(redisURL)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	cache, err := NewRedisCache(ctx, redisURL, redisTimeout)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, cases := getRedisCases()

	for _, c := range cases {
		err := cache.SetGroup(ctx, c.Key, c.Group)
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		res := miniredisClient.Get(ctx, c.Key)
		found := true
		err = res.Err()
		if err != nil {
			if err == redis.Nil {
				found = false
			} else {
				t.Errorf("Expected err to be nil but it was %q", err)
			}
		}

		var g models.Group

		err = res.Scan(&g)
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}

		if !cmp.Equal(c.Group, g) {
			t.Errorf("Expected response was not returned.\nExpected: %s\nActual:   %s", c.Group, g)
		}
		if err != nil {
			t.Errorf("Did not expect error: %q", err)
		}
		if !found {
			t.Errorf("Expected cached group to be found")
		}
	}

	redisServer.Close()
	err = cache.SetGroup(ctx, "no-redis-server", models.Group{})
	if !strings.Contains(err.Error(), "connect: connection refused") {
		t.Errorf("Expected error to contain 'connect: connection refused' but it was: %q", err)
	}
}

func getMiniredisClient(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opt), err
}

type redisUserCase struct {
	User models.User
	Key  string
}

type redisGroupCase struct {
	Group models.Group
	Key   string
}

func getRedisCases() ([]redisUserCase, []redisGroupCase) {
	userCases := []redisUserCase{
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

	groupCases := []redisGroupCase{
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
