package metrics

import (
	"context"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/internal/config"
	"github.com/xenitab/azad-kube-proxy/internal/models"
)

// ClientInterface ...
type ClientInterface interface {
	MetricsHandler(ctx context.Context, router *mux.Router) (*mux.Router, error)
}

// NewMetricsClient ...
func NewMetricsClient(ctx context.Context, cfg *config.Config) (ClientInterface, error) {
	metricsType, err := models.GetMetrics(cfg.Metrics)
	if err != nil {
		return nil, err
	}

	switch metricsType {
	case models.NoneMetrics:
		client := newNoneClient(ctx)
		return &client, nil
	case models.PrometheusMetrics:
		client := newPrometheusClient(ctx)
		return &client, nil
	default:
		return nil, fmt.Errorf("Unexpected metrics: %s", cfg.Metrics)
	}
}
