package utils

import (
	"errors"
	"fmt"
	"net/http"
)

type Code string

const (
	CodeInvalidArgument Code = "INVALID_ARGUMENT"
	CodeUnauthorized    Code = "UNAUTHORIZED"
	CodeForbidden       Code = "FORBIDDEN"
	CodeNotFound        Code = "NOT_FOUND"
	CodeConflict        Code = "CONFLICT"
	CodeUnavailable     Code = "UNAVAILABLE"
	CodeTimeout         Code = "TIMEOUT"
	CodeInternal        Code = "INTERNAL"
)

// AppError is the unified error contract across layers.
type AppError struct {
	Code    Code
	Op      string // operation name, ex: "SessionService.Start"
	Message string // safe message
	Err     error  // wrapped error
}

func (e *AppError) Error() string {
	if e == nil {
		return "<nil>"
	}
	switch {
	case e.Op != "" && e.Message != "" && e.Err != nil:
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
	case e.Op != "" && e.Message != "":
		return fmt.Sprintf("%s: %s", e.Op, e.Message)
	case e.Op != "" && e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	case e.Message != "" && e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	case e.Message != "":
		return e.Message
	case e.Err != nil:
		return e.Err.Error()
	default:
		return "error"
	}
}

func (e *AppError) Unwrap() error { return e.Err }

func E(code Code, op, msg string, err error) error {
	return &AppError{Code: code, Op: op, Message: msg, Err: err}
}

func IsCode(err error, code Code) bool {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code == code
	}
	return false
}

func HTTPStatus(err error) int {
	var ae *AppError
	if errors.As(err, &ae) {
		switch ae.Code {
		case CodeInvalidArgument:
			return http.StatusBadRequest
		case CodeUnauthorized:
			return http.StatusUnauthorized
		case CodeForbidden:
			return http.StatusForbidden
		case CodeNotFound:
			return http.StatusNotFound
		case CodeConflict:
			return http.StatusConflict
		case CodeUnavailable:
			return http.StatusServiceUnavailable
		case CodeTimeout:
			return http.StatusGatewayTimeout
		default:
			return http.StatusInternalServerError
		}
	}
	// fallback
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}

// Backward-compatible sentinel errors
var (
	ErrNotFound = errors.New("not found")
)
