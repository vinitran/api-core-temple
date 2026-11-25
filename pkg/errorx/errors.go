// Package errorx classifies application errors into kinds that map to
// transport-level status codes and client-safe messages.
package errorx

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ory/ladon"
)

// Kind is the category of an application error.
type Kind uint8

const (
	Other Kind = iota
	Invalid
	Validation
	NotExist
	Exist
	RateLimiting
	Authn
	Authz
	Captcha
	Database
	Service
	Unavailable
)

const pgUniqueViolation = "23505"

const maskedMessage = "unable to process"

func (k Kind) String() string {
	switch k {
	case Invalid:
		return "invalid-request"
	case Validation:
		return "validation"
	case NotExist:
		return "resource-not-found"
	case Exist:
		return "resource-already-exists"
	case RateLimiting:
		return "rate-limiting"
	case Authn:
		return "authentication"
	case Authz:
		return "authorization"
	case Captcha:
		return "captcha"
	case Database:
		return "database-query"
	case Service:
		return "internal-service-failure"
	case Unavailable:
		return "service-unavailable"
	}

	return "unknown"
}

// HTTPStatus maps the kind to an HTTP status code.
func (k Kind) HTTPStatus() int {
	switch k {
	case Invalid:
		return http.StatusBadRequest
	case Validation, Captcha:
		return http.StatusUnprocessableEntity
	case NotExist:
		return http.StatusNotFound
	case Exist:
		return http.StatusConflict
	case RateLimiting:
		return http.StatusTooManyRequests
	case Authn:
		return http.StatusUnauthorized
	case Authz:
		return http.StatusForbidden
	case Unavailable:
		return http.StatusServiceUnavailable
	}

	return http.StatusInternalServerError
}

// Error is an application error carrying a kind, a client-safe message and
// the underlying cause.
type Error struct {
	err     error
	kind    Kind
	message string
}

// Error returns the message together with the cause, intended for logs.
func (e *Error) Error() string {
	if e.err == nil || e.err.Error() == e.message {
		return e.message
	}

	return e.message + ": " + e.err.Error()
}

// Unwrap offers the ability to check the cause using errors.Is / errors.As.
func (e *Error) Unwrap() error {
	return e.err
}

// Of reports whether the error is of kind k.
func (e *Error) Of(k Kind) bool {
	return e.kind == k
}

// Kind returns the error kind.
func (e *Error) Kind() Kind {
	return e.kind
}

// Message returns the message without the underlying cause.
func (e *Error) Message() string {
	return e.message
}

// Code returns the machine-readable error code.
func (e *Error) Code() string {
	return e.kind.String()
}

// Status returns the HTTP status code.
func (e *Error) Status() int {
	return e.kind.HTTPStatus()
}

// New creates an error of the given kind without an underlying cause.
func New(kind Kind, message string) *Error {
	return &Error{kind: kind, message: message}
}

// Wrap classifies err. Errors that are already *Error keep their kind;
// well-known database / authorization errors are mapped to their natural kind;
// everything else gets the given kind. Wrap(nil, ...) returns nil.
func Wrap(err error, kind Kind) error {
	if err == nil {
		return nil
	}

	var target *Error
	if errors.As(err, &target) {
		return err
	}

	switch {
	case IsNoRows(err):
		return &Error{err: err, kind: NotExist, message: "resource not found"}
	case IsForbidden(err):
		return &Error{err: err, kind: Authz, message: "forbidden"}
	case IsDuplicated(err):
		return &Error{err: err, kind: Exist, message: "resource already exists"}
	case isPgError(err):
		return &Error{err: err, kind: Database, message: err.Error()}
	}

	return &Error{err: err, kind: kind, message: err.Error()}
}

// WrapMsg wraps err with an explicit kind and client-safe message.
// WrapMsg(nil, ...) returns nil.
func WrapMsg(err error, kind Kind, message string) error {
	if err == nil {
		return nil
	}

	return &Error{err: err, kind: kind, message: message}
}

// KindOf returns the kind of err, or Other when err is not an *Error.
func KindOf(err error) Kind {
	var target *Error
	if errors.As(err, &target) {
		return target.kind
	}

	return Other
}

func IsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows)
}

func IsForbidden(err error) bool {
	return errors.Is(err, ladon.ErrRequestDenied) || errors.Is(err, ladon.ErrRequestForcefullyDenied)
}

func IsDuplicated(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation
}

func isPgError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr)
}

// MaskErrorMessage returns a message that is safe to show to clients.
// Server-side failures (5xx) never leak their details.
func MaskErrorMessage(err error) string {
	var target *Error
	if !errors.As(err, &target) {
		return "unexpected error occurred"
	}

	if target.Status() >= http.StatusInternalServerError && !target.Of(Unavailable) {
		return maskedMessage
	}

	return target.message
}
