package models

import "fmt"

// Metrics ...
type Metrics string

// NoneMetrics ...
var NoneMetrics Metrics = "NONE"

// PrometheusMetrics ...
var PrometheusMetrics Metrics = "PROMETHEUS"

// GetMetrics ...
func GetMetrics(s string) (Metrics, error) {
	switch s {
	case "NONE":
		return NoneMetrics, nil
	case "PROMETHEUS":
		return PrometheusMetrics, nil
	default:
		return "", fmt.Errorf("Unknown metrics '%s'. Supported engines are: NONE or PROMETHEUS", s)
	}
}
