package proxy

import "fmt"

type metricsModel string

var noneMetrics metricsModel = "NONE"
var prometheusMetrics metricsModel = "PROMETHEUS"

func getMetrics(s string) (metricsModel, error) {
	switch s {
	case "NONE":
		return noneMetrics, nil
	case "PROMETHEUS":
		return prometheusMetrics, nil
	default:
		return "", fmt.Errorf("Unknown metrics '%s'. Supported engines are: NONE or PROMETHEUS", s)
	}
}
