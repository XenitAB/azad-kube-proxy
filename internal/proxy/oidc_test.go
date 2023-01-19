package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAzureADClaimsValidationFn(t *testing.T) {
	t.Run("no required tenant id", func(t *testing.T) {
		fn := newAzureADClaimsValidationFn("")
		err := fn(&externalAzureADClaims{})
		require.NoError(t, err)
	})

	t.Run("required tenant id with correct tid", func(t *testing.T) {
		fn := newAzureADClaimsValidationFn("ze-tenant")
		err := fn(&externalAzureADClaims{
			TenantId: testToPtr(t, "ze-tenant"),
		})
		require.NoError(t, err)
	})

	t.Run("required tenant id with missing tid", func(t *testing.T) {
		fn := newAzureADClaimsValidationFn("ze-tenant")
		err := fn(&externalAzureADClaims{})
		require.ErrorContains(t, err, "tid claim missing")
	})
	t.Run("required tenant id with wrong tid", func(t *testing.T) {
		fn := newAzureADClaimsValidationFn("ze-tenant")
		err := fn(&externalAzureADClaims{TenantId: testToPtr(t, "wrong-tenant")})
		require.ErrorContains(t, err, "tid claim is required to be \"ze-tenant\" but was: wrong-tenant")
	})
}

func testToPtr[P any](t *testing.T, v P) *P {
	t.Helper()
	return &v
}
