package cors

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

func TestMiddleware(t *testing.T) {
	fakeBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{\"fake\": true}"))
	}))
	defer fakeBackend.Close()
	fakeBackendURL, err := url.Parse(fakeBackend.URL)
	if err != nil {
		t.Errorf("Expected err to be nil but it was %q", err)
	}

	cases := []struct {
		config          config.Config
		reqHost         string
		reqOrigin       string
		reqMethod       string
		reqHeaders      map[string]string
		expectedOrigin  string
		expectedHeaders string
		expectedMethods string
	}{
		{
			config: config.Config{
				CORSConfig: config.CORSConfig{
					Enabled:                     true,
					AllowedOriginsDefaultScheme: "http",
					AllowedOrigins:              []string{},
					AllowedHeaders:              []string{},
					AllowedMethods:              []string{},
				},
			},
			reqHost:         "localhost",
			reqOrigin:       "http://localhost",
			reqMethod:       "GET",
			expectedOrigin:  "http://localhost",
			expectedHeaders: "",
			expectedMethods: "",
		},
		{
			config: config.Config{
				CORSConfig: config.CORSConfig{
					Enabled:                     true,
					AllowedOriginsDefaultScheme: "http",
					AllowedOrigins:              []string{},
					AllowedHeaders:              []string{},
					AllowedMethods:              []string{},
				},
			},
			reqHost:         "localhost",
			reqOrigin:       "http://localhost",
			reqMethod:       "OPTIONS",
			expectedOrigin:  "http://localhost",
			expectedHeaders: "",
			expectedMethods: "",
		},
		{
			config: config.Config{
				CORSConfig: config.CORSConfig{
					Enabled:                     false,
					AllowedOriginsDefaultScheme: "http",
					AllowedOrigins:              []string{},
					AllowedHeaders:              []string{},
					AllowedMethods:              []string{},
				},
			},
			reqHost:         "localhost",
			reqOrigin:       "http://localhost",
			reqMethod:       "OPTIONS",
			expectedOrigin:  "",
			expectedHeaders: "",
			expectedMethods: "",
		},
		{
			config: config.Config{
				CORSConfig: config.CORSConfig{
					Enabled:                     true,
					AllowedOriginsDefaultScheme: "http",
					AllowedOrigins:              []string{},
					AllowedHeaders:              []string{},
					AllowedMethods:              []string{},
				},
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
			config: config.Config{
				CORSConfig: config.CORSConfig{
					Enabled:                     true,
					AllowedOriginsDefaultScheme: "http",
					AllowedOrigins:              []string{},
					AllowedHeaders:              []string{"X-Fake-Header"},
					AllowedMethods:              []string{"GET"},
				},
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
			config: config.Config{
				CORSConfig: config.CORSConfig{
					Enabled:                     true,
					AllowedOriginsDefaultScheme: "http",
					AllowedOrigins:              []string{},
					AllowedHeaders:              []string{"X-Fake-Header"},
					AllowedMethods:              []string{"GET"},
				},
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
			config: config.Config{
				CORSConfig: config.CORSConfig{
					Enabled:                     true,
					AllowedOriginsDefaultScheme: "http",
					AllowedOrigins:              []string{},
					AllowedHeaders:              []string{"X-Fake-Header"},
					AllowedMethods:              []string{"PATCH"},
				},
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
			config: config.Config{
				CORSConfig: config.CORSConfig{
					Enabled:                     true,
					AllowedOriginsDefaultScheme: "http",
					AllowedOrigins:              []string{},
					AllowedHeaders:              []string{"X-Fake-Header", "Test-Header", "Abc-Header"},
					AllowedMethods:              []string{"POST", "GET"},
				},
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
			config: config.Config{
				CORSConfig: config.CORSConfig{
					Enabled:                     true,
					AllowedOriginsDefaultScheme: "http",
					AllowedOrigins:              []string{},
					AllowedHeaders:              []string{"X-Fake-Header", "Test-Header", "Abc-Header"},
					AllowedMethods:              []string{"POST", "GET"},
				},
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
		client := NewCORSClient(c.config)

		req, err := http.NewRequest(c.reqMethod, "/", nil)
		if err != nil {
			t.Errorf("Expected err to be nil but it was %q", err)
		}
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

		if rr.Result().Header.Get("Access-Control-Allow-Origin") != c.expectedOrigin {
			t.Errorf("unexpected Access-Control-Allow-Origin header: got %v want %v", rr.Result().Header.Get("Access-Control-Allow-Origin"), c.expectedOrigin)
		}

		if rr.Result().Header.Get("Access-Control-Allow-Headers") != c.expectedHeaders {
			t.Errorf("unexpected Access-Control-Allow-Headers header: got %v want %v", rr.Result().Header.Get("Access-Control-Allow-Headers"), c.expectedHeaders)
		}

		if rr.Result().Header.Get("Access-Control-Allow-Methods") != c.expectedMethods {
			t.Errorf("unexpected Access-Control-Allow-Methods header: got %v want %v", rr.Result().Header.Get("Access-Control-Allow-Methods"), c.expectedMethods)
		}
	}
}
