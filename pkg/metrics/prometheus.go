package metrics

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type prometheusClient struct{}

func newPrometheusClient(ctx context.Context) prometheusClient {
	log := logr.FromContext(ctx)
	log.Info("Using metrics: prometheus")

	return prometheusClient{}
}

func (client *prometheusClient) DashboardHandler(ctx context.Context, router *mux.Router) (*mux.Router, error) {
	return router, nil
}

// MetricsHandler ...
func (client *prometheusClient) MetricsHandler(ctx context.Context, router *mux.Router) (*mux.Router, error) {
	router.Handle("/metrics", promhttp.Handler())
	return router, nil
}
