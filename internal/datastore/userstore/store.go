// Package userstore persists users in Postgres.
package userstore

import (
	"context"
	"fmt"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"

	bobmodel "api-core/internal/bob"
	"api-core/internal/datastore"
	"api-core/internal/domain"
	"api-core/pkg/errorx"
)

// Store reads and writes users. Errors are classified with errorx
// (e.g. a missing row becomes errorx.NotExist).
type Store struct {
	exec bob.Executor
}

// New returns a store backed by the connection pool.
func New(pool datastore.PGXPool) *Store {
	return NewWithExecutor(datastore.NewBobExecutor(pool))
}

// NewWithExecutor returns a store bound to exec, e.g. a transaction executor
// passed to a datastore.TxFunc.
func NewWithExecutor(exec bob.Executor) *Store {
	return &Store{exec: exec}
}

// GetByID returns the user with the given id.
func (s *Store) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row, err := bobmodel.FindUser(ctx, s.exec, id)
	if err != nil {
		return nil, errorx.Wrap(fmt.Errorf("userstore: get by id %d: %w", id, err), errorx.Database)
	}

	return toDomain(row), nil
}

// GetByGoogleID returns the user linked to the given Google account.
func (s *Store) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	row, err := bobmodel.Users.Query(
		sm.Where(bobmodel.Users.Columns.GoogleID.EQ(psql.Arg(googleID))),
	).One(ctx, s.exec)
	if err != nil {
		return nil, errorx.Wrap(fmt.Errorf("userstore: get by google id: %w", err), errorx.Database)
	}

	return toDomain(row), nil
}

// UpsertGoogleUser creates the user on first sign-in and refreshes the
// profile on subsequent sign-ins, keyed by google_id.
func (s *Store) UpsertGoogleUser(ctx context.Context, acc domain.GoogleAccount) (*domain.User, error) {
	setter := &bobmodel.UserSetter{
		GoogleID:      omit.From(acc.GoogleID),
		Email:         omit.From(acc.Email),
		Name:          omitnull.FromPtr(acc.Name),
		Picture:       omitnull.FromPtr(acc.Picture),
		Locale:        omitnull.FromPtr(acc.Locale),
		VerifiedEmail: omitnull.From(acc.VerifiedEmail),
		UpdatedAt:     omit.From(acc.LoginAt),
		LastLoginAt:   omitnull.From(acc.LoginAt),
	}

	row, err := bobmodel.Users.Insert(
		setter,
		im.OnConflict(psql.Quote("google_id")).DoUpdate(
			im.SetExcluded("email", "name", "picture", "locale", "verified_email", "updated_at", "last_login_at"),
		),
	).One(ctx, s.exec)
	if err != nil {
		return nil, errorx.Wrap(fmt.Errorf("userstore: upsert google user: %w", err), errorx.Database)
	}

	return toDomain(row), nil
}

func toDomain(model *bobmodel.User) *domain.User {
	return &domain.User{
		ID:            model.ID,
		GoogleID:      model.GoogleID,
		Email:         model.Email,
		Name:          model.Name.Ptr(),
		Picture:       model.Picture.Ptr(),
		Locale:        model.Locale.Ptr(),
		VerifiedEmail: model.VerifiedEmail.GetOrZero(),
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
		LastLoginAt:   model.LastLoginAt.Ptr(),
	}
}
