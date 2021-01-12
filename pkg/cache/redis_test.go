package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-logr/logr"
	logrTesting "github.com/go-logr/logr/testing"
	"github.com/go-redis/redis/v8"
	"github.com/google/go-cmp/cmp"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

var (
	redisTimeout = 5 * time.Minute
)

func TestNewRedisCache(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})
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
}

func TestRedisGetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})
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

	_, found, _ := cache.GetUser(ctx, "does-not-exist")
	if found {
		t.Errorf("Expected cached user not to be found")
	}
}

func TestRedisSetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})
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
		cache.SetUser(ctx, c.Key, c.User)
		res := miniredisClient.Get(ctx, c.Key)
		found := true
		err := res.Err()
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
}

// func TestRedisGetGroup(t *testing.T) {
// 	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})
// 	redisServer, err := miniredis.Run()
// 	if err != nil {
// 		t.Errorf("Expected err to be nil but it was %q", err)
// 	}
// 	defer redisServer.Close()

// 	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
// 	miniredisClient, err := getMiniredisClient(redisURL)
// 	if err != nil {
// 		t.Errorf("Expected err to be nil but it was %q", err)
// 	}

// 	cache, err := NewRedisCache(ctx, redisURL, redisTimeout)
// 	if err != nil {
// 		t.Errorf("Expected err to be nil but it was %q", err)
// 	}

// 	_, cases := getRedisCases()

// 	for _, c := range cases {
// 		cache.Cache.Set(c.Key, c.Group, 0)
// 		cacheRes, found, err := cache.GetGroup(ctx, c.Key)
// 		if !cmp.Equal(c.Group, cacheRes) {
// 			t.Errorf("Expected response was not returned.\nExpected: %s\nActual:   %s", c.Group, cacheRes)
// 		}
// 		if err != nil {
// 			t.Errorf("Did not expect error: %q", err)
// 		}
// 		if !found {
// 			t.Errorf("Expected cached user to be found")
// 		}
// 	}

// 	_, found, _ := cache.GetGroup(ctx, "does-not-exist")
// 	if found {
// 		t.Errorf("Expected cached group not to be found")
// 	}
// }

// func TestRedisSetGroup(t *testing.T) {
// 	ctx := logr.NewContext(context.Background(), logrTesting.NullLogger{})
// 	redisServer, err := miniredis.Run()
// 	if err != nil {
// 		t.Errorf("Expected err to be nil but it was %q", err)
// 	}
// 	defer redisServer.Close()

// 	redisURL := fmt.Sprintf("redis://%s/0", redisServer.Addr())
// 	miniredisClient, err := getMiniredisClient(redisURL)
// 	if err != nil {
// 		t.Errorf("Expected err to be nil but it was %q", err)
// 	}

// 	cache, err := NewRedisCache(ctx, redisURL, redisTimeout)
// 	if err != nil {
// 		t.Errorf("Expected err to be nil but it was %q", err)
// 	}

// 	_, cases := getRedisCases()

// 	for _, c := range cases {
// 		cache.SetGroup(ctx, c.Key, c.Group)
// 		cacheRes, found := cache.Cache.Get(c.Key)
// 		if !cmp.Equal(c.Group, cacheRes) {
// 			t.Errorf("Expected response was not returned.\nExpected: %s\nActual:   %s", c.Group, cacheRes)
// 		}
// 		if err != nil {
// 			t.Errorf("Did not expect error: %q", err)
// 		}
// 		if !found {
// 			t.Errorf("Expected cached group to be found")
// 		}
// 	}
// }

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
