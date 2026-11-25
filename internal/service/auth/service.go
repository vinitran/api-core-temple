// Package auth implements sign-in with an external identity provider and
// issues the application's own access tokens.
package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"api-core/internal/domain"
	"api-core/pkg/errorx"
)

const defaultStateTTL = 5 * time.Minute

// ErrProviderDisabled is returned when no identity provider is configured.
var ErrProviderDisabled = errorx.New(errorx.Unavailable, "google sign-in is not configured")

// Deps are the collaborators of Service. Identity may be nil, in which case
// sign-in endpoints answer 503 while the rest of the API keeps working.
type Deps struct {
	Users    UserStore
	Identity IdentityProvider
	States   StateStore
	Tokens   TokenIssuer
}

type Service struct {
	users    UserStore
	identity IdentityProvider
	states   StateStore
	tokens   TokenIssuer

	stateTTL time.Duration
	now      func() time.Time
	newState func() string
}

// AuthResult is returned after a successful sign-in.
type AuthResult struct {
	AccessToken string
	ExpiresAt   time.Time
	User        *domain.User
}

func NewService(deps Deps) (*Service, error) {
	if deps.Users == nil || deps.States == nil || deps.Tokens == nil {
		return nil, errors.New("auth service: users, states and tokens are required")
	}

	return &Service{
		users:    deps.Users,
		identity: deps.Identity,
		states:   deps.States,
		tokens:   deps.Tokens,
		stateTTL: defaultStateTTL,
		now:      func() time.Time { return time.Now().UTC() },
		newState: uuid.NewString,
	}, nil
}

// GenerateLoginURL creates a one-time state and returns the provider's
// authorization URL.
func (s *Service) GenerateLoginURL(ctx context.Context) (string, error) {
	if s.identity == nil {
		return "", ErrProviderDisabled
	}

	state := s.newState()
	if err := s.states.Save(ctx, state, s.stateTTL); err != nil {
		return "", errorx.Wrap(err, errorx.Service)
	}

	return s.identity.AuthCodeURL(state), nil
}

// HandleCallback validates the state, exchanges the code, upserts the user and
// issues an access token.
func (s *Service) HandleCallback(ctx context.Context, state, code string) (*AuthResult, error) {
	if s.identity == nil {
		return nil, ErrProviderDisabled
	}
	if state == "" || code == "" {
		return nil, errorx.New(errorx.Invalid, "missing state or code")
	}

	ok, err := s.states.Consume(ctx, state)
	if err != nil {
		return nil, errorx.Wrap(err, errorx.Service)
	}
	if !ok {
		return nil, errorx.New(errorx.Authn, "invalid or expired state")
	}

	identity, err := s.identity.Authenticate(ctx, code)
	if err != nil {
		return nil, errorx.WrapMsg(err, errorx.Authn, "google authentication failed")
	}
	if identity.Subject == "" || identity.Email == "" {
		return nil, errorx.New(errorx.Authn, "google account has no id or email")
	}
	if !identity.EmailVerified {
		return nil, errorx.New(errorx.Authz, "google email is not verified")
	}

	user, err := s.users.UpsertGoogleUser(ctx, domain.GoogleAccount{
		GoogleID:      identity.Subject,
		Email:         identity.Email,
		Name:          nonEmpty(identity.Name),
		Picture:       nonEmpty(identity.Picture),
		Locale:        nonEmpty(identity.Locale),
		VerifiedEmail: identity.EmailVerified,
		LoginAt:       s.now(),
	})
	if err != nil {
		return nil, err
	}

	token, err := s.tokens.Issue(strconv.FormatInt(user.ID, 10), user.Email)
	if err != nil {
		return nil, errorx.Wrap(err, errorx.Service)
	}

	return &AuthResult{
		AccessToken: token.Value,
		ExpiresAt:   token.ExpiresAt,
		User:        user,
	}, nil
}

// CurrentUser returns the user identified by a verified token subject.
func (s *Service) CurrentUser(ctx context.Context, subject string) (*domain.User, error) {
	id, err := strconv.ParseInt(subject, 10, 64)
	if err != nil || id <= 0 {
		return nil, errorx.New(errorx.Authn, "invalid token subject")
	}

	return s.users.GetByID(ctx, id)
}

func nonEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	return &s
}
