package errorutil

// Code represents application-level error codes, transport-agnostic.
type Code int

const (
	CodeInternal      Code = iota // Unexpected error
	CodeNotFound                  // Resource not found
	CodeAlreadyExists             // Resource already exists
	CodeInvalidInput              // Validation failed
	CodeUnauthorized              // Authentication required
	CodeForbidden                 // Permission denied
	CodeConflict                  // State conflict
	CodePrecondition              // Precondition failed
)
