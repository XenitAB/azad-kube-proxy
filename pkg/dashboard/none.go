package dashboard

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
)

type noneClient struct{}

func newNoneClient(ctx context.Context) noneClient {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("Using dashboard: none")

	return noneClient{}
}

// DashboardHandler ...
func (client *noneClient) DashboardHandler(ctx context.Context, router *mux.Router) (*mux.Router, error) {
	return router, nil
}
