package handlers

import "fmt"

// azureClaims contains the claims used by the Azure AD Access Token (v2)
type azureClaims struct {
	sub      string
	username string
	objectID string
	groups   []string
}

// toAzureClaims converts the raw claims from context.Value() to azureClaims
func toAzureClaims(rawClaims map[string]interface{}) (azureClaims, error) {
	rawSub, ok := rawClaims["sub"]
	if !ok {
		return azureClaims{}, fmt.Errorf("unable to find sub claim")
	}

	sub, ok := rawSub.(string)
	if !ok {
		return azureClaims{}, fmt.Errorf("unable to typecast sub to string: %v", rawSub)
	}

	isServicePrincipal := false
	rawUsername, ok := rawClaims["preferred_username"]
	if !ok {
		isServicePrincipal = true
	}

	username := ""
	if !isServicePrincipal {
		username, ok = rawUsername.(string)
		if !ok {
			return azureClaims{}, fmt.Errorf("unable to typecast preferred_username to string: %v", rawUsername)
		}
	}

	rawObjectID, ok := rawClaims["oid"]
	if !ok {
		return azureClaims{}, fmt.Errorf("unable to find oid claim")
	}

	objectID, ok := rawObjectID.(string)
	if !ok {
		return azureClaims{}, fmt.Errorf("unable to typecast oid to string: %v", rawObjectID)
	}

	rawGroups := rawClaims["groups"]
	groups, ok := rawGroups.([]string)
	if !ok {
		// if we are unable to typecast, set groups to empty
		groups = []string{}
	}

	return azureClaims{
		sub:      sub,
		username: username,
		objectID: objectID,
		groups:   groups,
	}, nil
}
