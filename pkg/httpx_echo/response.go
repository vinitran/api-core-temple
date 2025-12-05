package httpx

import (
	"api-core/pkg/auth"
	"api-core/pkg/errorx"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/labstack/echo/v4"
)

type body struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func Abort(c echo.Context, v any, codes ...int) error {
	code := -1
	if len(codes) >= 1 {
		code = codes[0]
	}

	err, ok := v.(error)
	if ok {
		return abortErrorWithStatusJSON(c, err, code)
	}

	if code == -1 {
		code = 200
	}
	return c.JSON(code, &body{Data: v})
}

func abortErrorWithStatusJSON(c echo.Context, err error, code int) error {
	var target *errorx.Error

	message := errorx.MaskErrorMessage(err)

	if !errors.As(err, &target) {
		c.Logger().Error(err)
		if code == -1 {
			code = http.StatusInternalServerError
		}
		return c.JSON(code, &body{Code: "error", Message: message})
	}

	if code == -1 {
		code = target.Status()
	}

	if target.Of(errorx.Database) || target.Of(errorx.Service) {
		c.Logger().Error(err)
		return c.JSON(code, &body{Code: target.Code(), Message: message})
	}

	return c.JSON(code, &body{Code: target.Code(), Message: message})
}

type ValidatorStruct interface {
	Struct(s any) error
}

func ValidateStruct(c echo.Context, v ValidatorStruct, s any) error {
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	fields := []string{}
	for _, f := range validationErrors {
		fields = append(fields, f.StructField())
	}

	return fmt.Errorf("invalid %s", strings.Join(fields, ", "))
}

// RestAbort handles HTTP response with proper error handling.
// It returns a success response if err is nil, otherwise it wraps and returns the error.
// The function prioritizes specific error types (auth errors, errorx.Error) before generic errors.
func RestAbort(c echo.Context, v any, err error) error {
	// Success case: no error, return data
	if err == nil {
		return Abort(c, v)
	}

	// Handle specific authentication errors
	if errors.Is(err, auth.ErrInvalidSession) {
		return Abort(c, errorx.Wrap(err, errorx.Authn))
	}

	// Handle already wrapped errorx.Error (preserve the error kind)
	var targetErr *errorx.Error
	if errors.As(err, &targetErr) {
		return Abort(c, err)
	}

	// Generic error: wrap as Service error
	return Abort(c, errorx.Wrap(err, errorx.Service))
}
