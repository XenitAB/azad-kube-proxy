package proxy

import (
	"fmt"
	"net/http"

	rscors "github.com/rs/cors"
)

type Cors interface {
	Middleware(next http.Handler) http.Handler
}

type cors struct {
	enabled                     bool
	allowedOriginsDefaultScheme string
	allowedOrigins              []string
	allowedHeaders              []string
	allowedMethods              []string
}

func newCors(cfg *Config) *cors {
	allowedHeaders := cfg.CorsAllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"*"}
	}

	allowedMethods := cfg.CorsAllowedMethods
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE", "OPTIONS"}
	}

	return &cors{
		enabled:                     cfg.CorsEnabled,
		allowedOriginsDefaultScheme: cfg.CorsAllowedOriginsDefaultScheme,
		allowedOrigins:              cfg.CorsAllowedOrigins,
		allowedHeaders:              allowedHeaders,
		allowedMethods:              allowedMethods,
	}
}

// Middleware adds CORS to the router
func (c *cors) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.enabled {
			allowedOrigins := c.allowedOrigins
			if len(allowedOrigins) == 0 {
				url := fmt.Sprintf("%s://%s", c.allowedOriginsDefaultScheme, r.Host)
				allowedOrigins = []string{url}
			}

			c := rscors.New(rscors.Options{
				AllowedOrigins:   allowedOrigins,
				AllowedHeaders:   c.allowedHeaders,
				AllowedMethods:   c.allowedMethods,
				AllowCredentials: true,
			})

			corsHandler := c.Handler(next)
			corsHandler.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
