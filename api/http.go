package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/flanksource/commons/logger"
	"github.com/labstack/echo/v4"
	"github.com/samber/oops"
)

type HTTPError struct {
	Err     string `json:"error"`
	Message string `json:"message,omitempty"`

	// Data for machine-machine communication.
	// usually contains a JSON data.
	Data string `json:"data,omitempty"`
}

// Error implements the error interface. Not used by the application otherwise.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("error=%s message=%s data=%s ", e.Err, e.Message, e.Data)
}

func HTTPErrorFromErr(err error) *HTTPError {
	var e *HTTPError
	if errors.As(err, &e) {
		return e
	}

	return nil
}

type HTTPSuccess struct {
	Message string `json:"message"`
	Payload any    `json:"payload,omitempty"`
}

func WriteSuccess(c echo.Context, payload any) error {
	return c.JSON(http.StatusOK, HTTPSuccess{Message: "success", Payload: payload})
}

func WriteError(c echo.Context, err error) error {
	var oopsErr oops.OopsError
	if errors.As(err, &oopsErr) {
		code, _ := oopsErr.Code().(string)
		return c.JSON(ErrorStatusCode(code), oopsErr)
	}

	code, message, data := ErrorCode(err), ErrorMessage(err), ErrorData(err)

	if debugInfo := ErrorDebugInfo(err); debugInfo != "" {
		logger.WithValues("code", code, "error", message).Errorf(debugInfo)
	}

	return c.JSON(ErrorStatusCode(code), &HTTPError{Err: message, Data: data})
}

// ErrorStatusCode returns the associated HTTP status code for an application error code.
func ErrorStatusCode(code string) int {
	// lookup of application error codes to HTTP status codes.
	var codes = map[string]int{
		ECONFLICT:       http.StatusConflict,
		EINVALID:        http.StatusBadRequest,
		ENOTFOUND:       http.StatusNotFound,
		EFORBIDDEN:      http.StatusForbidden,
		ENOTIMPLEMENTED: http.StatusNotImplemented,
		EUNAUTHORIZED:   http.StatusUnauthorized,
		EINTERNAL:       http.StatusInternalServerError,
	}

	if v, ok := codes[code]; ok {
		return v
	}

	return http.StatusInternalServerError
}
