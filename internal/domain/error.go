package domain

import "fmt"

type ErrorKind string

const (
	ErrorValidation ErrorKind = "validation"
	ErrorDNS        ErrorKind = "dns"
	ErrorTimeout    ErrorKind = "timeout"
	ErrorRefused    ErrorKind = "refused"
	ErrorNetwork    ErrorKind = "network"
	ErrorProtocol   ErrorKind = "protocol"
	ErrorPayload    ErrorKind = "payload"
	ErrorCancelled  ErrorKind = "cancelled"
)

// AppError is stable across UI and protocol implementations. Cause is retained
// for diagnostics and errors.Is/errors.As, but must not be shown directly.
type AppError struct {
	Kind      ErrorKind
	Message   string
	Cause     error
	Retryable bool
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("mc server check failed (%s)", e.Kind)
}

func (e *AppError) Unwrap() error { return e.Cause }
