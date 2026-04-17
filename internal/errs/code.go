package errs

const (
	CodeOK                    = "OK"
	CodeBadRequest            = "BAD_REQUEST"
	CodeDependencyUnavailable = "DEPENDENCY_UNAVAILABLE"
	CodeInvalidCredentials    = "INVALID_CREDENTIALS"
	CodeUserAlreadyExists     = "USER_ALREADY_EXISTS"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeInternalError         = "INTERNAL_ERROR"
)

var defaultMessages = map[string]string{
	CodeOK:                    "success",
	CodeBadRequest:            "bad request",
	CodeDependencyUnavailable: "dependency unavailable",
	CodeInvalidCredentials:    "invalid username or password",
	CodeUserAlreadyExists:     "user already exists",
	CodeUnauthorized:          "unauthorized",
	CodeInternalError:         "internal server error",
}

func DefaultMessage(code string) string {
	if message, ok := defaultMessages[code]; ok {
		return message
	}

	return "unknown error"
}
