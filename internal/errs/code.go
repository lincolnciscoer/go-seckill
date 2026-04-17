package errs

const (
	CodeOK                    = "OK"
	CodeBadRequest            = "BAD_REQUEST"
	CodeDependencyUnavailable = "DEPENDENCY_UNAVAILABLE"
	CodeActivityInactive      = "ACTIVITY_INACTIVE"
	CodeActivityNotStarted    = "ACTIVITY_NOT_STARTED"
	CodeActivityEnded         = "ACTIVITY_ENDED"
	CodeDuplicateOrder        = "DUPLICATE_ORDER"
	CodeInvalidCredentials    = "INVALID_CREDENTIALS"
	CodeOrderNotFound         = "ORDER_NOT_FOUND"
	CodeSoldOut               = "SOLD_OUT"
	CodeUserAlreadyExists     = "USER_ALREADY_EXISTS"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeInternalError         = "INTERNAL_ERROR"
)

var defaultMessages = map[string]string{
	CodeOK:                    "success",
	CodeActivityInactive:      "activity inactive",
	CodeActivityNotStarted:    "activity not started",
	CodeActivityEnded:         "activity ended",
	CodeBadRequest:            "bad request",
	CodeDependencyUnavailable: "dependency unavailable",
	CodeDuplicateOrder:        "duplicate order",
	CodeInvalidCredentials:    "invalid username or password",
	CodeOrderNotFound:         "order not found",
	CodeSoldOut:               "sold out",
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
