package handlers

import (
	"fmt"
	"net/http"

	"github.com/xenitab/go-oidc-middleware/oidchttp"
	"github.com/xenitab/go-oidc-middleware/options"
)

// NewOIDCHandler returns a http.Handler to take care of the JWT validation
func NewOIDCHandler(h http.HandlerFunc, tenantID string, clientID string) http.Handler {
	oidcHandler := oidchttp.New(h,
		options.WithIssuer(fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)),
		options.WithRequiredTokenType("JWT"),
		options.WithRequiredAudience(clientID),
		options.WithFallbackSignatureAlgorithm("RS256"),
		options.WithRequiredClaims(map[string]interface{}{
			"tid": tenantID,
		}),
		options.WithLazyLoadJwks(true),
	)

	return oidcHandler
}
