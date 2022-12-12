package handlers

import "fmt"

// azureClaims contains the claims used by the Azure AD Access Token (v2)
type internalAzureADClaims struct {
	sub      string
	username string
	objectID string
	groups   []string
}

// toInternalAzureADClaims converts the externalAzureADClaims from context.Value() to externalAzureADClaims
func toInternalAzureADClaims(externalClaims externalAzureADClaims) (internalAzureADClaims, error) {
	if externalClaims.Subject == nil {
		return internalAzureADClaims{}, fmt.Errorf("unable to find sub claim")
	}
	subject := *externalClaims.Subject

	if externalClaims.ObjectId == nil {
		return internalAzureADClaims{}, fmt.Errorf("unable to find oid claim")
	}
	objectId := *externalClaims.ObjectId

	username := ""
	if externalClaims.PreferredUsername != nil {
		username = *externalClaims.PreferredUsername
	}

	groups := []string{}
	if externalClaims.Groups != nil {
		groups = *externalClaims.Groups
	}

	return internalAzureADClaims{
		sub:      subject,
		username: username,
		objectID: objectId,
		groups:   groups,
	}, nil
}
