package customerrors

type ErrorType string

const (
	ErrorTypeUnknown         ErrorType = "Unknown"
	ErrorTypeAuthentication  ErrorType = "Authentication"
	ErrorTypeAuthorization   ErrorType = "Authorization"
	ErrorTypeKubeConfig      ErrorType = "KubeConfig"
	ErrorTypeTokenCache      ErrorType = "TokenCache"
	ErrorTypeCACertificate   ErrorType = "CACertificate"
	ErrorTypeOverwriteConfig ErrorType = "OverwriteConfig"
	ErrorTypeMenu            ErrorType = "Menu"
)

type CustomError struct {
	ErrorType    ErrorType
	ErrorMessage string
}

func (ce *CustomError) Error() string {
	switch ce.ErrorType {
	default:
		return "Unknown error"
	case ErrorTypeUnknown:
		return "Unknown error"
	case ErrorTypeAuthentication:
		return "Authentication error: Please validate that you are logged on using the correct credentials"
	case ErrorTypeAuthorization:
		return "Authorization error: You don't have the privileges required to perform the action"
	case ErrorTypeKubeConfig:
		return "Kube config error: Unable to load file"
	case ErrorTypeTokenCache:
		return "Token cache error: Please validate that you have write access to the token cache path"
	case ErrorTypeCACertificate:
		return "CA certificate error: Unable to get CA certificate from server - validate that you have access to the proxy"
	case ErrorTypeOverwriteConfig:
		return "Overwrite config error: Overwrite config (--overwrite) is not enabled"
	case ErrorTypeMenu:
		return "Menu error: Unable to run the menu"
	}
}

func New(et ErrorType, e error) *CustomError {
	return &CustomError{
		ErrorType:    et,
		ErrorMessage: e.Error(),
	}
}

func To(e error) *CustomError {
	result, ok := e.(*CustomError)
	if !ok {
		result = &CustomError{
			ErrorType:    ErrorTypeUnknown,
			ErrorMessage: "Non-default error",
		}
	}
	return result
}
