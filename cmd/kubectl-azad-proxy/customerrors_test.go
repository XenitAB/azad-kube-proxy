package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	var fakeerrorType errorType = "fake"
	cases := []struct {
		customError   *customError
		errorContains string
	}{
		{
			customError:   newCustomError(errorTypeUnknown, errors.New("Dummy")),
			errorContains: "Unknown",
		},
		{
			customError:   newCustomError(fakeerrorType, errors.New("Dummy")),
			errorContains: "Unknown",
		},
		{
			customError:   newCustomError(errorTypeAuthentication, errors.New("Dummy")),
			errorContains: "Authentication error: ",
		},
		{
			customError:   newCustomError(errorTypeAuthorization, errors.New("Dummy")),
			errorContains: "Authorization error: ",
		},
		{
			customError:   newCustomError(errorTypeKubeConfig, errors.New("Dummy")),
			errorContains: "Kube config error: ",
		},
		{
			customError:   newCustomError(errorTypeTokenCache, errors.New("Dummy")),
			errorContains: "Token cache error: ",
		},
		{
			customError:   newCustomError(errorTypeCACertificate, errors.New("Dummy")),
			errorContains: "CA certificate error: ",
		},
		{
			customError:   newCustomError(errorTypeOverwriteConfig, errors.New("Dummy")),
			errorContains: "Overwrite config error: ",
		},
		{
			customError:   newCustomError(errorTypeMenu, errors.New("Dummy")),
			errorContains: "Menu error: ",
		},
	}

	for _, c := range cases {
		require.ErrorContains(t, c.customError, c.errorContains)
	}
}

func TestNewCustomError(t *testing.T) {
	ce := newCustomError(errorTypeUnknown, errors.New("Unknown"))
	require.Equal(t, errorTypeUnknown, ce.errorType)
	require.Equal(t, "Unknown", ce.errorMessage)
}

func TestCustomError(t *testing.T) {
	a := &customError{
		errorType:    errorTypeUnknown,
		errorMessage: "Unknown",
	}

	ce := toCustomError(a)
	require.Equal(t, errorTypeUnknown, ce.errorType)
	require.Equal(t, "Unknown", ce.errorMessage)

	b := errors.New("Fake")
	ce = toCustomError(b)
	require.Equal(t, errorTypeUnknown, ce.errorType)
	require.Equal(t, "Non-default error", ce.errorMessage)
}
