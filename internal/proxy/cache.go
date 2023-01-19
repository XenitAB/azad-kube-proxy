package proxy

import (
	"context"
)

type Cache interface {
	getUser(ctx context.Context, s string) (userModel, bool, error)
	setUser(ctx context.Context, s string, u userModel) error
	getGroup(ctx context.Context, s string) (groupModel, bool, error)
	setGroup(ctx context.Context, s string, g groupModel) error
}
