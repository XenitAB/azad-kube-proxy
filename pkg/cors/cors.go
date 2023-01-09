package cors

import (
	"fmt"
	"net/http"

	"github.com/rs/cors"
	"github.com/xenitab/azad-kube-proxy/pkg/config"
)

// ClientInterface ...
type ClientInterface interface {
	Middleware(next http.Handler) http.Handler
}

// Client ...
type Client struct {
	enabled                     bool
	allowedOriginsDefaultScheme string
	allowedOrigins              []string
	allowedHeaders              []string
	allowedMethods              []string
}

// NewCORSClient ...
func NewCORSClient(config *config.Config) ClientInterface {
	allowedHeaders := config.CorsAllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"*"}
	}

	allowedMethods := config.CorsAllowedMethods
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE", "OPTIONS"}
	}

	return &Client{
		enabled:                     config.CorsEnabled,
		allowedOriginsDefaultScheme: config.CorsAllowedOriginsDefaultScheme,
		allowedOrigins:              config.CorsAllowedOrigins,
		allowedHeaders:              allowedHeaders,
		allowedMethods:              allowedMethods,
	}
}

// Middleware adds CORS to the router
func (client *Client) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if client.enabled {
			allowedOrigins := client.allowedOrigins
			if len(allowedOrigins) == 0 {
				url := fmt.Sprintf("%s://%s", client.allowedOriginsDefaultScheme, r.Host)
				allowedOrigins = []string{url}
			}

			c := cors.New(cors.Options{
				AllowedOrigins:   allowedOrigins,
				AllowedHeaders:   client.allowedHeaders,
				AllowedMethods:   client.allowedMethods,
				AllowCredentials: true,
			})

			corsHandler := c.Handler(next)
			corsHandler.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
