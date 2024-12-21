package auth

import (
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/ory/ladon"
)

type AuthzAction string

const (
	DeleteAuthzAction AuthzAction = "delete"
	ReadAuthzAction   AuthzAction = "read"
)

type Guard struct {
	authn AuthnChecker
	authz AuthzChecker
}

type AuthzChecker interface {
	IsAllowed(r *ladon.Request) error
}

type AuthnChecker interface {
	AuthenticateJWT(tokenStr string) (*jwt.Token, error)
}

func NewGuard(authn AuthnChecker, authz AuthzChecker) (*Guard, error) {
	if authn == nil || authz == nil {
		return nil, errors.New("invalid authn or authz")
	}

	return &Guard{authn, authz}, nil
}

func (guard *Guard) Allow(sub string, resource string, action AuthzAction, ctx map[string]any) error {
	r := &ladon.Request{
		Subject:  sub,
		Resource: resource,
		Action:   string(action),
		Context:  ctx,
	}

	return guard.authz.IsAllowed(r)
}

func (g Guard) AuthenticateJWT(tokenStr string) (*jwt.Token, error) {
	return g.authn.AuthenticateJWT(tokenStr)
}
