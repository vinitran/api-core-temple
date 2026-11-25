package auth

import (
	"errors"

	"github.com/ory/ladon"
)

type AuthzAction string

const (
	DeleteAuthzAction AuthzAction = "delete"
	ReadAuthzAction   AuthzAction = "read"
)

// AuthzChecker decides whether an access request is allowed.
// *ladon.Ladon satisfies it.
type AuthzChecker interface {
	IsAllowed(r *ladon.Request) error
}

// Guard performs authorization checks. Authentication is the job of the
// token verifier used by the HTTP middleware.
type Guard struct {
	authz AuthzChecker
}

func NewGuard(authz AuthzChecker) (*Guard, error) {
	if authz == nil {
		return nil, errors.New("auth: nil authz checker")
	}

	return &Guard{authz: authz}, nil
}

// Allow returns nil when sub may perform action on resource.
func (g *Guard) Allow(sub, resource string, action AuthzAction, ctx map[string]any) error {
	return g.authz.IsAllowed(&ladon.Request{
		Subject:  sub,
		Resource: resource,
		Action:   string(action),
		Context:  ctx,
	})
}
