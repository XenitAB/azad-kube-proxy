package main

type errorType string

const (
	errorTypeUnknown         errorType = "Unknown"
	errorTypeAuthentication  errorType = "Authentication"
	errorTypeAuthorization   errorType = "Authorization"
	errorTypeKubeConfig      errorType = "KubeConfig"
	errorTypeTokenCache      errorType = "TokenCache"
	errorTypeCACertificate   errorType = "CACertificate"
	errorTypeOverwriteConfig errorType = "OverwriteConfig"
	errorTypeMenu            errorType = "Menu"
)

type customError struct {
	errorType    errorType
	errorMessage string
}

func (ce *customError) Error() string {
	switch ce.errorType {
	default:
		return "Unknown error"
	case errorTypeUnknown:
		return "Unknown error"
	case errorTypeAuthentication:
		return "Authentication error: Please validate that you are logged on using the correct credentials"
	case errorTypeAuthorization:
		return "Authorization error: You don't have the privileges required to perform the action"
	case errorTypeKubeConfig:
		return "Kube config error: Unable to load file"
	case errorTypeTokenCache:
		return "Token cache error: Please validate that you have write access to the token cache path"
	case errorTypeCACertificate:
		return "CA certificate error: Unable to get CA certificate from server - validate that you have access to the proxy"
	case errorTypeOverwriteConfig:
		return "Overwrite config error: Overwrite config (--overwrite) is not enabled"
	case errorTypeMenu:
		return "Menu error: Unable to run the menu"
	}
}

func newCustomError(et errorType, e error) *customError {
	return &customError{
		errorType:    et,
		errorMessage: e.Error(),
	}
}

func toCustomError(e error) *customError {
	result, ok := e.(*customError)
	if !ok {
		result = &customError{
			errorType:    errorTypeUnknown,
			errorMessage: "Non-default error",
		}
	}
	return result
}
