package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryCache(t *testing.T) {
	_, err := newMemoryCache(5 * time.Minute)
	require.NoError(t, err)
}

func TestMemoryGetUser(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cache, err := newMemoryCache(5 * time.Minute)
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
	cache, err := newMemoryCache(5 * time.Minute)
	require.NoError(t, err)

	cases, _ := testGetMemoryCases(t)

	for _, c := range cases {
		err := cache.SetUser(ctx, c.Key, c.User)
		require.NoError(t, err)

		cacheRes, found := cache.CacheClient.Get(c.Key)
		require.True(t, found)
		require.Equal(t, c.User, cacheRes.(userModel))
	}
}

func TestMemoryGetGroup(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	cache, err := newMemoryCache(5 * time.Minute)
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
	cache, err := newMemoryCache(5 * time.Minute)
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
		if !cmp.Equal(c.Group, cacheRes.(groupModel)) {
			t.Errorf("Expected response was not returned.\nExpected: %s\nActual:   %s", c.Group, cacheRes.(groupModel))
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
	User userModel
	Key  string
}

type testMemoryGroupCase struct {
	Group groupModel
	Key   string
}

func testGetMemoryCases(t *testing.T) ([]testMemoryUserCase, []testMemoryGroupCase) {
	t.Helper()

	userCases := []testMemoryUserCase{
		{
			User: userModel{
				Username: "user1",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []groupModel{
					{
						Name: "group1",
					},
				},
				Type: normalUserModelType,
			},
			Key: "tokenHash1",
		},
		{
			User: userModel{
				Username: "user2",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []groupModel{
					{
						Name: "group2",
					},
				},
				Type: normalUserModelType,
			},
			Key: "tokenHash2",
		},
		{
			User: userModel{
				Username: "00000000-0000-0000-0000-000000000000",
				ObjectID: "00000000-0000-0000-0000-000000000000",
				Groups: []groupModel{
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
				Type: servicePrincipalUserModelType,
			},
			Key: "tokenHash3",
		},
	}

	groupCases := []testMemoryGroupCase{
		{
			Group: groupModel{Name: "group1"},
			Key:   "00000000-0000-0000-0000-000000000000",
		},
		{
			Group: groupModel{Name: "group2"},
			Key:   "00000000-0000-0000-0000-000000000001",
		},
		{
			Group: groupModel{Name: "group3"},
			Key:   "00000000-0000-0000-0000-000000000002",
		},
	}

	return userCases, groupCases
}
