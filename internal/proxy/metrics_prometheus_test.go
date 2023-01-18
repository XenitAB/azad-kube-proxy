package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestPrometheusMetricsHandler(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	req, err := http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)

	client := newPrometheusClient(ctx)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router, err = client.MetricsHandler(ctx, router)
	require.NoError(t, err)
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.")
}
