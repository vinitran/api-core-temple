package auth

import (
	"context"
	"time"

	"api-core/internal/domain"
	"api-core/pkg/jwtx"
)

// The service depends only on these narrow interfaces (owned by the consumer),
// never on concrete stores or SDKs. Implementations are wired in the
// container, and tests substitute fakes.

// UserStore persists users. Implemented by *userstore.Store.
type UserStore interface {
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	UpsertGoogleUser(ctx context.Context, acc domain.GoogleAccount) (*domain.User, error)
}

// IdentityProvider runs the external OAuth2 authorization-code flow.
// Implemented by *GoogleProvider.
type IdentityProvider interface {
	AuthCodeURL(state string) string
	Authenticate(ctx context.Context, code string) (*Identity, error)
}

// StateStore keeps one-time OAuth states. Implemented by *oauthstate.RedisStore.
type StateStore interface {
	Save(ctx context.Context, state string, ttl time.Duration) error
	Consume(ctx context.Context, state string) (bool, error)
}

// TokenIssuer issues the application's access tokens. Implemented by *jwtx.HMAC.
type TokenIssuer interface {
	Issue(subject, email string) (*jwtx.Token, error)
}

// Identity is the profile returned by an IdentityProvider.
type Identity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
	Locale        string
}
