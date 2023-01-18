package proxy

import (
	"context"
)

type Cache interface {
	GetUser(ctx context.Context, s string) (userModel, bool, error)
	SetUser(ctx context.Context, s string, u userModel) error
	GetGroup(ctx context.Context, s string) (groupModel, bool, error)
	SetGroup(ctx context.Context, s string, g groupModel) error
}
