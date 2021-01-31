package dashboard

import (
	"context"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// ClientInterface ...
type ClientInterface interface {
	DashboardHandler(ctx context.Context, router *mux.Router) *mux.Router
}

// NewDashboardClient ...
func NewDashboardClient(ctx context.Context, config config.Config) (ClientInterface, error) {
	switch config.Dashboard {
	case models.NoneDashboard:
		client := newNoneClient(ctx)
		return &client, nil
	case models.K8sdashDashboard:
		client, err := newK8sdashClient(ctx, config)
		if err != nil {
			return nil, err
		}

		return &client, nil
	default:
		return nil, fmt.Errorf("Unexpected dashboard: %s", config.Dashboard)
	}
}
