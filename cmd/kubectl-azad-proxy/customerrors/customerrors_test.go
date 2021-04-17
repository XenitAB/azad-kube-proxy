package customerrors

import (
	"errors"
	"strings"
	"testing"
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
		if strings.Contains(c.errorContains, c.customError.Error()) {
			t.Errorf("Expected customError to contain %q but was: %q", c.errorContains, c.customError.Error())
		}
	}
}

func TestNew(t *testing.T) {
	ce := New(ErrorTypeUnknown, errors.New("Unknown"))
	if ce.ErrorType != ErrorTypeUnknown {
		t.Errorf("Expected ErrorType to be ErrorTypeUnknown: %q", ce.ErrorType)
	}

	if ce.ErrorMessage != "Unknown" {
		t.Errorf("Expected ErrorMessage to be 'Unknown': %q", ce.ErrorMessage)
	}
}

func TestTo(t *testing.T) {
	a := &CustomError{
		ErrorType:    ErrorTypeUnknown,
		ErrorMessage: "Unknown",
	}

	ce := To(a)
	if ce.ErrorType != ErrorTypeUnknown {
		t.Errorf("Expected ErrorType to be ErrorTypeUnknown: %q", ce.ErrorType)
	}

	if ce.ErrorMessage != "Unknown" {
		t.Errorf("Expected ErrorMessage to be 'Unknown': %q", ce.ErrorMessage)
	}

	b := errors.New("Fake")
	ce = To(b)
	if ce.ErrorType != ErrorTypeUnknown {
		t.Errorf("Expected ErrorType to be ErrorTypeUnknown: %q", ce.ErrorType)
	}

	if ce.ErrorMessage != "Non-default error" {
		t.Errorf("Expected ErrorMessage to be 'Non-default error': %q", ce.ErrorMessage)
	}
}
