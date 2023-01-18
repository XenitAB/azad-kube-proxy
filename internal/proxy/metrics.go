package proxy

import (
	"context"
	"fmt"

	"github.com/gorilla/mux"
)

type ClientInterface interface {
	MetricsHandler(ctx context.Context, router *mux.Router) (*mux.Router, error)
}

func NewMetricsClient(ctx context.Context, cfg *Config) (ClientInterface, error) {
	metricsType, err := getMetrics(cfg.Metrics)
	if err != nil {
		return nil, err
	}

	switch metricsType {
	case noneMetrics:
		client := newNoneClient(ctx)
		return &client, nil
	case prometheusMetrics:
		client := newPrometheusClient(ctx)
		return &client, nil
	default:
		return nil, fmt.Errorf("Unexpected metrics: %s", cfg.Metrics)
	}
}
