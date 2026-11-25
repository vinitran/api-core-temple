package auth

import (
	"context"
	"errors"

	"api-core/pkg/jwtx"
)

type ctxKey int

const (
	ctxKeyAuthClaims ctxKey = iota
	ctxKeyAuthJWT
)

// ErrInvalidSession is returned when the request carries no authenticated subject.
var ErrInvalidSession = errors.New("invalid session")

// WithAuthClaims stores verified token claims in ctx.
func WithAuthClaims(ctx context.Context, claims *jwtx.Claims) context.Context {
	return context.WithValue(ctx, ctxKeyAuthClaims, claims)
}

// WithAuthJWT stores the raw bearer token in ctx.
func WithAuthJWT(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, ctxKeyAuthJWT, token)
}

// ResolveJWT returns the raw bearer token, or "" when absent.
func ResolveJWT(ctx context.Context) string {
	token, _ := ctx.Value(ctxKeyAuthJWT).(string)
	return token
}

// ResolveClaims returns the verified claims stored in ctx.
func ResolveClaims(ctx context.Context) (*jwtx.Claims, bool) {
	claims, ok := ctx.Value(ctxKeyAuthClaims).(*jwtx.Claims)
	return claims, ok && claims != nil
}

// ResolveSubject returns the authenticated subject, or "" when absent.
func ResolveSubject(ctx context.Context) string {
	claims, ok := ResolveClaims(ctx)
	if !ok {
		return ""
	}

	return claims.Subject
}

// ResolveEmail returns the authenticated email, or "" when absent.
func ResolveEmail(ctx context.Context) string {
	claims, ok := ResolveClaims(ctx)
	if !ok {
		return ""
	}

	return claims.Email
}

// ResolveValidSubject returns the authenticated subject or ErrInvalidSession.
func ResolveValidSubject(ctx context.Context) (string, error) {
	sub := ResolveSubject(ctx)
	if sub == "" {
		return "", ErrInvalidSession
	}

	return sub, nil
}
