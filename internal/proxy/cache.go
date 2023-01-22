package proxy

import (
	"context"
)

type cacheReadWriter interface {
	userCacheReader
	userCacheWriter
	groupCacheReader
	groupCacheWriter
}

type userCacheReader interface {
	getUser(ctx context.Context, s string) (userModel, bool, error)
}

type userCacheWriter interface {
	setUser(ctx context.Context, s string, u userModel) error
}

type groupCacheReader interface {
	getGroup(ctx context.Context, s string) (groupModel, bool, error)
}

type groupCacheWriter interface {
	setGroup(ctx context.Context, s string, g groupModel) error
}
