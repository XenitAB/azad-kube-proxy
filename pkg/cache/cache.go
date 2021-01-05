package cache

import (
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// Client ...
type Client interface {
	GetUser(s string) (models.User, bool)
	SetUser(s string, u models.User) error
	GetGroup(s string) (models.Group, bool)
	SetGroup(s string, g models.Group) error
	NewCache()
}
