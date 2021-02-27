package metrics

import (
	"context"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
	"github.com/xenitab/azad-kube-proxy/pkg/models"
)

// ClientInterface ...
type ClientInterface interface {
	MetricsHandler(ctx context.Context, router *mux.Router) (*mux.Router, error)
}

// NewMetricsClient ...
func NewMetricsClient(ctx context.Context, config config.Config) (ClientInterface, error) {
	switch config.Metrics {
	case models.NoneMetrics:
		client := newNoneClient(ctx)
		return &client, nil
	case models.PrometheusMetrics:
		client := newPrometheusClient(ctx)
		return &client, nil
	default:
		return nil, fmt.Errorf("Unexpected metrics: %s", config.Metrics)
	}
}
