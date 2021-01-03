package claims

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
	Username       string       `json:"preferred_username"`
	Groups         []string     `json:"groups"`
}
