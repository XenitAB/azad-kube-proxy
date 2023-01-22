package proxy

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	rscors "github.com/rs/cors"
)

func newCorsMiddleware(cfg *config) mux.MiddlewareFunc {
	allowedHeaders := cfg.CorsAllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"*"}
	}

	allowedMethods := cfg.CorsAllowedMethods
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE", "OPTIONS"}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.CorsEnabled {
				allowedOrigins := cfg.CorsAllowedOrigins
				if len(allowedOrigins) == 0 {
					url := fmt.Sprintf("%s://%s", cfg.CorsAllowedOriginsDefaultScheme, r.Host)
					allowedOrigins = []string{url}
				}

				c := rscors.New(rscors.Options{
					AllowedOrigins:   allowedOrigins,
					AllowedHeaders:   cfg.CorsAllowedHeaders,
					AllowedMethods:   cfg.CorsAllowedMethods,
					AllowCredentials: true,
				})

				corsHandler := c.Handler(next)
				corsHandler.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

}