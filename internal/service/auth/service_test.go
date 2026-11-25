package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"api-core/internal/domain"
	"api-core/pkg/errorx"
	"api-core/pkg/jwtx"
)

type fakeUsers struct {
	upserted []domain.GoogleAccount
	byID     map[int64]*domain.User
}

func (f *fakeUsers) GetByID(_ context.Context, id int64) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, errorx.New(errorx.NotExist, "resource not found")
	}
	return u, nil
}

func (f *fakeUsers) UpsertGoogleUser(_ context.Context, acc domain.GoogleAccount) (*domain.User, error) {
	f.upserted = append(f.upserted, acc)
	return &domain.User{ID: 7, GoogleID: acc.GoogleID, Email: acc.Email, Name: acc.Name}, nil
}

type fakeIdentity struct {
	identity *Identity
	err      error
}

func (f *fakeIdentity) AuthCodeURL(state string) string {
	return "https://idp.example/auth?state=" + state
}

func (f *fakeIdentity) Authenticate(context.Context, string) (*Identity, error) {
	return f.identity, f.err
}

type fakeStates struct {
	saved map[string]bool
	err   error
}

func (f *fakeStates) Save(_ context.Context, state string, _ time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.saved[state] = true
	return nil
}

func (f *fakeStates) Consume(_ context.Context, state string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	ok := f.saved[state]
	delete(f.saved, state)
	return ok, nil
}

type fakeTokens struct{}

func (fakeTokens) Issue(subject, _ string) (*jwtx.Token, error) {
	return &jwtx.Token{Value: "token-for-" + subject}, nil
}

type fixture struct {
	svc      *Service
	users    *fakeUsers
	identity *fakeIdentity
	states   *fakeStates
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	f := &fixture{
		users: &fakeUsers{byID: map[int64]*domain.User{7: {ID: 7, Email: "a@example.com"}}},
		identity: &fakeIdentity{identity: &Identity{
			Subject: "g-1", Email: "a@example.com", EmailVerified: true, Name: "Alice",
		}},
		states: &fakeStates{saved: map[string]bool{}},
	}

	svc, err := NewService(Deps{Users: f.users, Identity: f.identity, States: f.states, Tokens: fakeTokens{}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	svc.newState = func() string { return "state-1" }
	f.svc = svc

	return f
}

func assertKind(t *testing.T, err error, want errorx.Kind) {
	t.Helper()

	if got := errorx.KindOf(err); got != want {
		t.Fatalf("error kind = %v (%v), want %v", got, err, want)
	}
}

func TestLoginAndCallback(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	ctx := context.Background()

	url, err := f.svc.GenerateLoginURL(ctx)
	if err != nil {
		t.Fatalf("GenerateLoginURL() error = %v", err)
	}
	if url != "https://idp.example/auth?state=state-1" {
		t.Errorf("url = %q", url)
	}

	res, err := f.svc.HandleCallback(ctx, "state-1", "code")
	if err != nil {
		t.Fatalf("HandleCallback() error = %v", err)
	}
	if res.AccessToken != "token-for-7" || res.User.ID != 7 {
		t.Errorf("result = %+v", res)
	}
	if len(f.users.upserted) != 1 || f.users.upserted[0].Locale != nil {
		t.Errorf("upserted = %+v, want one account with nil locale", f.users.upserted)
	}

	// State is single use: replaying the callback must fail.
	_, err = f.svc.HandleCallback(ctx, "state-1", "code")
	assertKind(t, err, errorx.Authn)
}

func TestHandleCallbackErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup func(f *fixture)
		state string
		code  string
		want  errorx.Kind
	}{
		"missing code": {
			state: "state-1", code: "", want: errorx.Invalid,
		},
		"unknown state": {
			state: "other", code: "code", want: errorx.Authn,
		},
		"provider failure": {
			setup: func(f *fixture) { f.identity.err = errors.New("exchange failed") },
			state: "state-1", code: "code", want: errorx.Authn,
		},
		"unverified email": {
			setup: func(f *fixture) { f.identity.identity.EmailVerified = false },
			state: "state-1", code: "code", want: errorx.Authz,
		},
		"state store down": {
			setup: func(f *fixture) { f.states.err = errors.New("redis down") },
			state: "state-1", code: "code", want: errorx.Service,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.states.saved["state-1"] = true
			if tt.setup != nil {
				tt.setup(f)
			}

			_, err := f.svc.HandleCallback(context.Background(), tt.state, tt.code)
			assertKind(t, err, tt.want)
		})
	}
}

func TestProviderDisabled(t *testing.T) {
	t.Parallel()

	svc, err := NewService(Deps{Users: &fakeUsers{}, States: &fakeStates{}, Tokens: fakeTokens{}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = svc.GenerateLoginURL(context.Background())
	assertKind(t, err, errorx.Unavailable)
}

func TestCurrentUser(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	user, err := f.svc.CurrentUser(context.Background(), "7")
	if err != nil || user.ID != 7 {
		t.Fatalf("CurrentUser() = %+v, %v", user, err)
	}

	_, err = f.svc.CurrentUser(context.Background(), "not-a-number")
	assertKind(t, err, errorx.Authn)

	_, err = f.svc.CurrentUser(context.Background(), "99")
	assertKind(t, err, errorx.NotExist)
}
