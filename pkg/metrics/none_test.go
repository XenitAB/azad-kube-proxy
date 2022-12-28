package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestNoneMetricsHandler(t *testing.T) {
	ctx := logr.NewContext(context.Background(), logr.Discard())
	req, err := http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)

	client := newNoneClient(ctx)
	require.NoError(t, err)

	fakeBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{\"fake\": true}"))
	}))
	defer fakeBackend.Close()
	fakeBackendURL, err := url.Parse(fakeBackend.URL)
	require.NoError(t, err)

	proxy := httputil.NewSingleHostReverseProxy(fakeBackendURL)
	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.PathPrefix("/").Handler(proxy)
	router, err = client.MetricsHandler(ctx, router)
	require.NoError(t, err)
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), "{\"fake\": true}")
}
