package api

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/flanksource/commons/logger"
)

// Application error codes.
//
// These are meant to be generic and they map well to HTTP error codes.
const (
	ECONFLICT       = "conflict"
	EFORBIDDEN      = "forbidden"
	EINTERNAL       = "internal"
	EINVALID        = "invalid"
	ENOTFOUND       = "not_found"
	ENOTIMPLEMENTED = "not_implemented"
	EUNAUTHORIZED   = "unauthorized"
)

// Error represents an application-specific error.
type Error struct {
	// Machine-readable error code.
	Code string

	// Human-readable error message.
	Message string

	// Machine-machine error message.
	Data string

	// DebugInfo contains low-level internal error details that should only be logged.
	// End-users should never see this.
	DebugInfo string
}

// Error implements the error interface. Not used by the application otherwise.
func (e *Error) Error() string {
	return fmt.Sprintf("error: code=%s message=%s", e.Code, e.Message)
}

// WithDebugInfo wraps an application error with a debug message.
func (e *Error) WithDebugInfo(msg string, args ...any) *Error {
	e.DebugInfo = fmt.Sprintf(msg, args...)
	return e
}

// WithData sets the given data
func (e *Error) WithData(data any) *Error {
	switch v := data.(type) {
	case string:
		e.Data = v
	default:
		d, err := json.Marshal(data)
		if err != nil {
			logger.Errorf("failed to set data. json marshalling failed: %v", err)
		} else {
			e.Data = string(d)
		}
	}

	return e
}

// ErrorCode unwraps an application error and returns its code.
// Non-application errors always return EINTERNAL.
func ErrorCode(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Code
	}
	return EINTERNAL
}

// ErrorMessage unwraps an application error and returns its message.
func ErrorMessage(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Message
	}

	return err.Error()
}

// ErrorMessage unwraps an application error and returns its message.
// Non-application errors always return "Internal error".
func ErrorData(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.Data
	}
	return ""
}

// ErrorDebugInfo unwraps an application error and returns its debug message.
func ErrorDebugInfo(err error) string {
	var e *Error
	if err == nil {
		return ""
	} else if errors.As(err, &e) {
		return e.DebugInfo
	}

	return err.Error()
}

// Errorf is a helper function to return an Error with a given code and formatted message.
func Errorf(code string, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

func FromError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}

	return nil
}
