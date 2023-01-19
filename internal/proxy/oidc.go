package proxy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/xenitab/go-oidc-middleware/oidchttp"
	"github.com/xenitab/go-oidc-middleware/options"
)

func newOIDCHandler(h http.HandlerFunc, tenantID string, clientID string) http.Handler {
	oidcHandler := oidchttp.New(h,
		newAzureADClaimsValidationFn(tenantID),
		options.WithIssuer(fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)),
		options.WithRequiredTokenType("JWT"),
		options.WithRequiredAudience(clientID),
		options.WithFallbackSignatureAlgorithm("RS256"),
		options.WithLazyLoadJwks(true),
	)

	return oidcHandler
}

type externalAzureADClaims struct {
	Aio               *string    `json:"aio"`
	Audience          *[]string  `json:"aud"`
	Azpacr            *string    `json:"azpacr"`
	Azp               *string    `json:"azp"`
	ExpiresAt         *time.Time `json:"exp"`
	Groups            *[]string  `json:"groups"`
	Idp               *string    `json:"idp"`
	IssuedAt          *time.Time `json:"iat"`
	Issuer            *string    `json:"iss"`
	Name              *string    `json:"name"`
	NotBefore         *time.Time `json:"nbf"`
	ObjectId          *string    `json:"oid"`
	PreferredUsername *string    `json:"preferred_username"`
	Rh                *string    `json:"rh"`
	Scope             *string    `json:"scp"`
	Subject           *string    `json:"sub"`
	TenantId          *string    `json:"tid"`
	TokenVersion      *string    `json:"ver"`
	Uti               *string    `json:"uti"`
}

func newAzureADClaimsValidationFn(requiredTenantId string) options.ClaimsValidationFn[externalAzureADClaims] {
	return func(claims *externalAzureADClaims) error {
		if requiredTenantId == "" {
			return nil
		}

		if claims.TenantId == nil {
			return fmt.Errorf("tid claim missing")
		}

		if *claims.TenantId != requiredTenantId {
			return fmt.Errorf("tid claim is required to be %q but was: %s", requiredTenantId, *claims.TenantId)
		}

		return nil
	}
}
