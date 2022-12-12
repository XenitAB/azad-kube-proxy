package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToInternalAzureADClaims(t *testing.T) {
	t.Run("external claims nil", func(t *testing.T) {
		_, err := toInternalAzureADClaims(nil)
		require.ErrorContains(t, err, "external claims nil")
	})

	t.Run("subject nil", func(t *testing.T) {
		_, err := toInternalAzureADClaims(&externalAzureADClaims{})
		require.ErrorContains(t, err, "unable to find sub claim")
	})

	t.Run("object id nil", func(t *testing.T) {
		_, err := toInternalAzureADClaims(&externalAzureADClaims{
			Subject: testToPtr(t, "ze-subject"),
		})
		require.ErrorContains(t, err, "unable to find oid claim")
	})

	t.Run("username nil", func(t *testing.T) {
		internalClaims, err := toInternalAzureADClaims(&externalAzureADClaims{
			Subject:  testToPtr(t, "ze-subject"),
			ObjectId: testToPtr(t, "ze-object-id"),
		})
		require.NoError(t, err)
		require.Equal(t, "ze-subject", internalClaims.sub)
		require.Equal(t, "ze-object-id", internalClaims.objectID)
		require.Empty(t, internalClaims.username)
		require.Empty(t, internalClaims.groups)
	})

	t.Run("groups nil", func(t *testing.T) {
		internalClaims, err := toInternalAzureADClaims(&externalAzureADClaims{
			Subject:           testToPtr(t, "ze-subject"),
			ObjectId:          testToPtr(t, "ze-object-id"),
			PreferredUsername: testToPtr(t, "ze-username"),
		})
		require.NoError(t, err)
		require.Equal(t, "ze-subject", internalClaims.sub)
		require.Equal(t, "ze-object-id", internalClaims.objectID)
		require.Equal(t, "ze-username", internalClaims.username)
		require.Empty(t, internalClaims.groups)
	})

	t.Run("every parameter has a value", func(t *testing.T) {
		internalClaims, err := toInternalAzureADClaims(&externalAzureADClaims{
			Subject:           testToPtr(t, "ze-subject"),
			ObjectId:          testToPtr(t, "ze-object-id"),
			PreferredUsername: testToPtr(t, "ze-username"),
			Groups:            testToPtr(t, []string{"ze-group"}),
		})
		require.NoError(t, err)
		require.Equal(t, "ze-subject", internalClaims.sub)
		require.Equal(t, "ze-object-id", internalClaims.objectID)
		require.Equal(t, "ze-username", internalClaims.username)
		require.Equal(t, []string{"ze-group"}, internalClaims.groups)
	})
}
