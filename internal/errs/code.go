package errs

const (
	CodeOK                    = "OK"
	CodeBadRequest            = "BAD_REQUEST"
	CodeDependencyUnavailable = "DEPENDENCY_UNAVAILABLE"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeInternalError         = "INTERNAL_ERROR"
)

var defaultMessages = map[string]string{
	CodeOK:                    "success",
	CodeBadRequest:            "bad request",
	CodeDependencyUnavailable: "dependency unavailable",
	CodeUnauthorized:          "unauthorized",
	CodeInternalError:         "internal server error",
}

func DefaultMessage(code string) string {
	if message, ok := defaultMessages[code]; ok {
		return message
	}

	return "unknown error"
}
