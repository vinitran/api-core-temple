package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"

	"api-core/pkg/auth"
	"api-core/pkg/errorx"
)

// Body is the envelope of every JSON response.
type Body struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// Success writes data with 200 OK.
func Success(c echo.Context, data any) error {
	return c.JSON(http.StatusOK, &Body{Data: data})
}

// Fail writes err as an error response. The status and code come from the
// errorx kind; errors without a kind are treated as internal failures.
// Server-side failures are logged and their details are never sent to clients.
func Fail(c echo.Context, err error) error {
	err = normalize(err)

	var target *errorx.Error
	if !errors.As(err, &target) {
		// unreachable: normalize always returns *errorx.Error
		return c.JSON(http.StatusInternalServerError, &Body{Code: "error", Message: errorx.MaskErrorMessage(err)})
	}

	status := target.Status()
	if status >= http.StatusInternalServerError {
		req := c.Request()
		slog.ErrorContext(req.Context(), "request failed",
			"method", req.Method,
			"path", req.URL.Path,
			"status", status,
			"error", err,
		)
	}

	return c.JSON(status, &Body{Code: target.Code(), Message: errorx.MaskErrorMessage(err)})
}

// RestAbort writes v on success, or err as an error response.
func RestAbort(c echo.Context, v any, err error) error {
	if err != nil {
		return Fail(c, err)
	}

	return Success(c, v)
}

// normalize gives every error a kind.
func normalize(err error) error {
	if errors.Is(err, auth.ErrInvalidSession) {
		return errorx.WrapMsg(err, errorx.Authn, "authentication required")
	}

	return errorx.Wrap(err, errorx.Service)
}

// StructValidator validates structs; *validator.Validate satisfies it.
type StructValidator interface {
	Struct(s any) error
}

// ValidateStruct validates s and returns a Validation error naming the
// offending fields.
func ValidateStruct(v StructValidator, s any) error {
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return errorx.Wrap(err, errorx.Invalid)
	}

	fields := make([]string, 0, len(validationErrors))
	for _, f := range validationErrors {
		fields = append(fields, f.Field())
	}

	return errorx.WrapMsg(err, errorx.Validation, fmt.Sprintf("invalid %s", strings.Join(fields, ", ")))
}

// ErrorHandler is an echo.HTTPErrorHandler that renders framework errors
// (404, 405, body limit, ...) with the same envelope as application errors.
func ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var he *echo.HTTPError
	if !errors.As(err, &he) {
		_ = Fail(c, err)
		return
	}

	code := strings.ToLower(strings.ReplaceAll(http.StatusText(he.Code), " ", "-"))
	message := http.StatusText(he.Code)
	if msg, ok := he.Message.(string); ok && he.Code < http.StatusInternalServerError {
		message = msg
	}

	if c.Request().Method == http.MethodHead {
		_ = c.NoContent(he.Code)
		return
	}

	_ = c.JSON(he.Code, &Body{Code: code, Message: message})
}
