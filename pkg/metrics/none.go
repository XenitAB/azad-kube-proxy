package metrics

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
)

type noneClient struct{}

func newNoneClient(ctx context.Context) noneClient {
	log := logr.FromContext(ctx)
	log.Info("Using metrics: none")

	return noneClient{}
}

// MetricsHandler ...
func (client *noneClient) MetricsHandler(ctx context.Context, router *mux.Router) (*mux.Router, error) {
	return router, nil
}
