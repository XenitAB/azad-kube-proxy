package cache

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/xenitab/azad-kube-proxy/internal/models"
)

func TestNewMemoryCache(t *testing.T) {
	_, err := NewMemoryCache(5 * time.Minute)
	require.NoError(t, err)
}

func TestMemoryGetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cache, err := NewMemoryCache(5 * time.Minute)
	require.NoError(t, err)

	cases, _ := testGetMemoryCases(t)

	for _, c := range cases {
		cache.CacheClient.Set(c.Key, c.User, 0)
		cacheRes, found, err := cache.GetUser(ctx, c.Key)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, c.User, cacheRes)
	}

	_, found, _ := cache.GetUser(ctx, "does-not-exist")
	require.False(t, found)
}

func TestMemorySetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cache, err := NewMemoryCache(5 * time.Minute)
	require.NoError(t, err)

	cases, _ := testGetMemoryCases(t)

	for _, c := range cases {
		err := cache.SetUser(ctx, c.Key, c.User)
		require.NoError(t, err)

		cacheRes, found := cache.CacheClient.Get(c.Key)
		require.True(t, found)
		require.Equal(t, c.User, cacheRes.(models.User))
	}
}

func TestMemoryGetGroup(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cache, err := NewMemoryCache(5 * time.Minute)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, cases := testGetMemoryCases(t)

	for _, c := range cases {
		cache.CacheClient.Set(c.Key, c.Group, 0)
		cacheRes, found, err := cache.GetGroup(ctx, c.Key)

		require.NoError(t, err)
		require.Equal(t, c.Group, cacheRes)
		require.True(t, found)
	}

	_, found, _ := cache.GetGroup(ctx, "does-not-exist")
	require.False(t, found)
}

func TestMemorySetGroup(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cache, err := NewMemoryCache(5 * time.Minute)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	_, cases := testGetMemoryCases(t)

	for _, c := range cases {
		err := cache.SetGroup(ctx, c.Key, c.Group)
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}
		cacheRes, found := cache.CacheClient.Get(c.Key)
		if !cmp.Equal(c.Group, cacheRes.(models.Group)) {
			t.Errorf("Expected response was not returned.\nExpected: %s\nActual:   %s", c.Group, cacheRes.(models.Group))
		}
		if err != nil {
			t.Errorf("Did not expect error: %q", err)
		}
		if !found {
			t.Errorf("Expected cached group to be found")
		}
	}
}

type testMemoryUserCase struct {
	User models.User
	Key  string
}

type testMemoryGroupCase struct {
	Group models.Group
	Key   string
}

func testGetMemoryCases(t *testing.T) ([]testMemoryUserCase, []testMemoryGroupCase) {
	t.Helper()

	userCases := []testMemoryUserCase{
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

	groupCases := []testMemoryGroupCase{
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
