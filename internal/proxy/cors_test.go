package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	fakeBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{\"fake\": true}"))
	}))
	defer fakeBackend.Close()
	fakeBackendURL, err := url.Parse(fakeBackend.URL)
	require.NoError(t, err)

	cases := []struct {
		config          *config
		reqHost         string
		reqOrigin       string
		reqMethod       string
		reqHeaders      map[string]string
		expectedOrigin  string
		expectedHeaders string
		expectedMethods string
	}{
		{
			config: &config{
				CorsEnabled:                     true,
				CorsAllowedOriginsDefaultScheme: "http",
				CorsAllowedOrigins:              []string{},
				CorsAllowedHeaders:              []string{},
				CorsAllowedMethods:              []string{},
			},
			reqHost:         "localhost",
			reqOrigin:       "http://localhost",
			reqMethod:       "GET",
			expectedOrigin:  "http://localhost",
			expectedHeaders: "",
			expectedMethods: "",
		},
		{
			config: &config{
				CorsEnabled:                     true,
				CorsAllowedOriginsDefaultScheme: "http",
				CorsAllowedOrigins:              []string{},
				CorsAllowedHeaders:              []string{},
				CorsAllowedMethods:              []string{},
			},
			reqHost:         "localhost",
			reqOrigin:       "http://localhost",
			reqMethod:       "OPTIONS",
			expectedOrigin:  "http://localhost",
			expectedHeaders: "",
			expectedMethods: "",
		},
		{
			config: &config{
				CorsEnabled:                     false,
				CorsAllowedOriginsDefaultScheme: "http",
				CorsAllowedOrigins:              []string{},
				CorsAllowedHeaders:              []string{},
				CorsAllowedMethods:              []string{},
			},
			reqHost:         "localhost",
			reqOrigin:       "http://localhost",
			reqMethod:       "OPTIONS",
			expectedOrigin:  "",
			expectedHeaders: "",
			expectedMethods: "",
		},
		{
			config: &config{
				CorsEnabled:                     true,
				CorsAllowedOriginsDefaultScheme: "http",
				CorsAllowedOrigins:              []string{},
				CorsAllowedHeaders:              []string{},
				CorsAllowedMethods:              []string{},
			},
			reqHost:   "localhost",
			reqOrigin: "http://localhost",
			reqMethod: "OPTIONS",
			reqHeaders: map[string]string{
				"Access-Control-Request-Method":  "GET",
				"Access-Control-Request-Headers": "X-Fake-Header",
			},
			expectedOrigin:  "http://localhost",
			expectedHeaders: "X-Fake-Header",
			expectedMethods: "GET",
		},
		{
			config: &config{
				CorsEnabled:                     true,
				CorsAllowedOriginsDefaultScheme: "http",
				CorsAllowedOrigins:              []string{},
				CorsAllowedHeaders:              []string{"X-Fake-Header"},
				CorsAllowedMethods:              []string{"GET"},
			},
			reqHost:   "localhost",
			reqOrigin: "http://localhost",
			reqMethod: "OPTIONS",
			reqHeaders: map[string]string{
				"Access-Control-Request-Method":  "GET",
				"Access-Control-Request-Headers": "X-Fake-Header",
			},
			expectedOrigin:  "http://localhost",
			expectedHeaders: "X-Fake-Header",
			expectedMethods: "GET",
		},
		{
			config: &config{
				CorsEnabled:                     true,
				CorsAllowedOriginsDefaultScheme: "http",
				CorsAllowedOrigins:              []string{},
				CorsAllowedHeaders:              []string{"X-Fake-Header"},
				CorsAllowedMethods:              []string{"GET"},
			},
			reqHost:   "localhost",
			reqOrigin: "http://localhost",
			reqMethod: "GET",
			reqHeaders: map[string]string{
				"X-Fake-Header": "TEST",
			},
			expectedOrigin:  "http://localhost",
			expectedHeaders: "",
			expectedMethods: "",
		},
		{
			config: &config{
				CorsEnabled:                     true,
				CorsAllowedOriginsDefaultScheme: "http",
				CorsAllowedOrigins:              []string{},
				CorsAllowedHeaders:              []string{"X-Fake-Header"},
				CorsAllowedMethods:              []string{"PATCH"},
			},
			reqHost:   "localhost",
			reqOrigin: "http://localhost",
			reqMethod: "GET",
			reqHeaders: map[string]string{
				"X-Fake-Header": "TEST",
			},
			expectedOrigin:  "",
			expectedHeaders: "",
			expectedMethods: "",
		},
		{
			config: &config{
				CorsEnabled:                     true,
				CorsAllowedOriginsDefaultScheme: "http",
				CorsAllowedOrigins:              []string{},
				CorsAllowedHeaders:              []string{"X-Fake-Header", "Test-Header", "Abc-Header"},
				CorsAllowedMethods:              []string{"POST", "GET"},
			},
			reqHost:   "localhost",
			reqOrigin: "http://localhost",
			reqMethod: "GET",
			reqHeaders: map[string]string{
				"Abc-Header": "TEST",
			},
			expectedOrigin:  "http://localhost",
			expectedHeaders: "",
			expectedMethods: "",
		},
		{
			config: &config{
				CorsEnabled:                     true,
				CorsAllowedOriginsDefaultScheme: "http",
				CorsAllowedOrigins:              []string{},
				CorsAllowedHeaders:              []string{"X-Fake-Header", "Test-Header", "Abc-Header"},
				CorsAllowedMethods:              []string{"POST", "GET"},
			},
			reqHost:   "localhost",
			reqOrigin: "http://localhost",
			reqMethod: "DELETE",
			reqHeaders: map[string]string{
				"Abc-Header": "TEST",
			},
			expectedOrigin:  "",
			expectedHeaders: "",
			expectedMethods: "",
		},
	}

	for _, c := range cases {
		client := newCors(c.config)

		req, err := http.NewRequest(c.reqMethod, "/", nil)
		require.NoError(t, err)
		req.Host = c.reqHost
		req.Header.Add("Origin", c.reqOrigin)
		for k, v := range c.reqHeaders {
			req.Header.Add(k, v)
		}

		proxy := httputil.NewSingleHostReverseProxy(fakeBackendURL)

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.Handle("/", proxy)
		router.Use(client.Middleware)
		router.ServeHTTP(rr, req)

		require.Equal(t, c.expectedOrigin, rr.Result().Header.Get("Access-Control-Allow-Origin"))
		require.Equal(t, c.expectedHeaders, rr.Result().Header.Get("Access-Control-Allow-Headers"))
		require.Equal(t, c.expectedMethods, rr.Result().Header.Get("Access-Control-Allow-Methods"))
	}
}
