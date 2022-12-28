package customerrors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	var fakeErrorType ErrorType = "fake"
	cases := []struct {
		customError   *CustomError
		errorContains string
	}{
		{
			customError:   New(ErrorTypeUnknown, errors.New("Dummy")),
			errorContains: "Unknown",
		},
		{
			customError:   New(fakeErrorType, errors.New("Dummy")),
			errorContains: "Unknown",
		},
		{
			customError:   New(ErrorTypeAuthentication, errors.New("Dummy")),
			errorContains: "Authentication error: ",
		},
		{
			customError:   New(ErrorTypeAuthorization, errors.New("Dummy")),
			errorContains: "Authorization error: ",
		},
		{
			customError:   New(ErrorTypeKubeConfig, errors.New("Dummy")),
			errorContains: "Kube config error: ",
		},
		{
			customError:   New(ErrorTypeTokenCache, errors.New("Dummy")),
			errorContains: "Token cache error: ",
		},
		{
			customError:   New(ErrorTypeCACertificate, errors.New("Dummy")),
			errorContains: "CA certificate error: ",
		},
		{
			customError:   New(ErrorTypeOverwriteConfig, errors.New("Dummy")),
			errorContains: "Overwrite config error: ",
		},
		{
			customError:   New(ErrorTypeMenu, errors.New("Dummy")),
			errorContains: "Menu error: ",
		},
	}

	for _, c := range cases {
		require.ErrorContains(t, c.customError, c.errorContains)
	}
}

func TestNew(t *testing.T) {
	ce := New(ErrorTypeUnknown, errors.New("Unknown"))
	require.Equal(t, ErrorTypeUnknown, ce.ErrorType)
	require.Equal(t, "Unknown", ce.ErrorMessage)
}

func TestTo(t *testing.T) {
	a := &CustomError{
		ErrorType:    ErrorTypeUnknown,
		ErrorMessage: "Unknown",
	}

	ce := To(a)
	require.Equal(t, ErrorTypeUnknown, ce.ErrorType)
	require.Equal(t, "Unknown", ce.ErrorMessage)

	b := errors.New("Fake")
	ce = To(b)
	require.Equal(t, ErrorTypeUnknown, ce.ErrorType)
	require.Equal(t, "Non-default error", ce.ErrorMessage)
}
