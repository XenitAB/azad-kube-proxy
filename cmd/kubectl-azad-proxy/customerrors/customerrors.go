package customerrors

type ErrorType string

const (
	ErrorTypeUnknown    ErrorType = "Unknown"
	ErrorTypeAuth       ErrorType = "Auth"
	ErrorTypeTokenCache ErrorType = "TokenCache"
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
	case ErrorTypeAuth:
		return "Auth error: Please validate that you are logged on using the correct credentials"
	case ErrorTypeTokenCache:
		return "Token cache error: Please validate that you have write access to the token cache path"
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
