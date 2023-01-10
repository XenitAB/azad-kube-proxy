package cache

import (
	"context"

	"github.com/xenitab/azad-kube-proxy/internal/models"
)

// ClientInterface ...
type ClientInterface interface {
	GetUser(ctx context.Context, s string) (models.User, bool, error)
	SetUser(ctx context.Context, s string, u models.User) error
	GetGroup(ctx context.Context, s string) (models.Group, bool, error)
	SetGroup(ctx context.Context, s string, g models.Group) error
}
