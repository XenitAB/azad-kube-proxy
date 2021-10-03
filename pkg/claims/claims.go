package claims

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc"
	"github.com/go-logr/logr"
)

// ClaimNames contains the _claim_names struct
type ClaimNames struct {
	Groups string `json:"groups"`
}

// ClaimSourcesSource contains the src1 struct
// TODO: Could there be something else than Endpoint?
type ClaimSourcesSource struct {
	Endpoint string `json:"endpoint"`
}

// ClaimSources contains _claim_sources struct
// TODO: Could there be more than one source?
type ClaimSources struct {
	Source1 ClaimSourcesSource `json:"src1"`
}

// AzureClaims contains the Azure AD v2 token claims
type AzureClaims struct {
	Audience       string       `json:"aud"`
	Issuer         string       `json:"iss"`
	IssuedAt       int64        `json:"iat"`
	NotBefore      int64        `json:"nbf"`
	ExpirationTime int64        `json:"exp"`
	ClaimNames     ClaimNames   `json:"_claim_names"`
	ClaimSources   ClaimSources `json:"_claim_sources"`
	Subject        string       `json:"sub"`
	TokenVersion   string       `json:"ver"`
	TenantID       string       `json:"tid"`
	ApplicationID  string       `json:"azp"`
	ObjectID       string       `json:"oid"`
	Username       string       `json:"preferred_username"`
	Groups         []string     `json:"groups"`
}

// ClientInterface ...
type ClientInterface interface {
	NewClaims(t *oidc.IDToken) (AzureClaims, error)
	GetOIDCVerifier(ctx context.Context, tenantID, clientID string) (*oidc.IDTokenVerifier, error)
}

// Client ...
type Client struct{}

// NewClaimsClient ...
func NewClaimsClient() ClientInterface {
	return &Client{}
}

// NewClaims returns AzureClaims
func (client *Client) NewClaims(t *oidc.IDToken) (AzureClaims, error) {
	var c AzureClaims

	if t == nil {
		return AzureClaims{}, errors.New("Token nil")
	}

	err := t.Claims(&c)
	if err != nil {
		return AzureClaims{}, err
	}

	return c, nil
}

// GetOIDCVerifier returns an ID Token Verifier or an error
func (client *Client) GetOIDCVerifier(ctx context.Context, tenantID, clientID string) (*oidc.IDTokenVerifier, error) {
	log := logr.FromContextOrDiscard(ctx)
	issuerURL := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		log.Error(err, "Unable to initiate OIDC provider")
		return nil, err
	}

	oidcConfig := &oidc.Config{
		ClientID: clientID,
	}

	verifier := provider.Verifier(oidcConfig)

	return verifier, nil
}
