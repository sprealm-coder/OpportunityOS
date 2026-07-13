package platform

import "fmt"

type Error struct {
	Code    string
	Message string
	Field   string
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

func Invalid(code, message string) error { return &Error{Code: code, Message: message} }

var (
	ErrTenantRequired         = Invalid("tenant_required", "tenant scope is required")
	ErrPermissionDenied       = Invalid("permission_denied", "actor lacks the required permission")
	ErrInvalidTransition      = Invalid("invalid_transition", "state transition is not allowed")
	ErrIdempotencyKeyRequired = Invalid("idempotency_key_required", "idempotency key is required")
)
