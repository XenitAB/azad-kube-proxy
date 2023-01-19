package proxy

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricsRequestsCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "azad_kube_proxy_request_count",
		Help: "Total number of successful requests to azad-kube-proxy",
	}, []string{"kubectl_version"})
)

func incrementRequestCount(req *http.Request) {
	kubectlVersion := userAgentToKubectlVersion(req.Header.Get("User-Agent"))
	metricsRequestsCount.With(prometheus.Labels{
		"kubectl_version": kubectlVersion,
	}).Inc()
}

func userAgentToKubectlVersion(userAgent string) string {
	parts := strings.SplitN(userAgent, " ", 20)
	for _, part := range parts {
		if strings.Contains(part, "kubectl/") {
			return strings.Replace(part, "kubectl/", "", 1)
		}
	}

	return "unknown"
}
