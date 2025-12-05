package userstore

import (
	"context"
	"fmt"
	"time"

	bobmodel "api-core/internal/bob"
	"api-core/internal/datastore"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/expr"
)

type Store interface {
	GetByGoogleID(ctx context.Context, googleID string) (*User, error)
	UpsertGoogleUser(ctx context.Context, params UpsertGoogleUserParams) (*User, error)
}

type store struct {
	exec bob.Executor
}

func New(pool datastore.PGXPool) Store {
	return NewWithExecutor(datastore.NewBobExecutor(pool))
}

func NewWithExecutor(exec bob.Executor) Store {
	return &store{
		exec: exec,
	}
}

type UpsertGoogleUserParams struct {
	GoogleID      string
	Email         string
	Name          *string
	Picture       *string
	Locale        *string
	VerifiedEmail bool
	LoginAt       time.Time
}

type User struct {
	ID            int64
	GoogleID      string
	Email         string
	Name          *string
	Picture       *string
	Locale        *string
	VerifiedEmail bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastLoginAt   *time.Time
}

func (s *store) GetByGoogleID(ctx context.Context, googleID string) (*User, error) {
	row, err := bobmodel.Users.Query(
		sm.Where(bobmodel.Users.Columns.GoogleID.EQ(psql.Arg(googleID))),
	).One(ctx, s.exec)
	if err != nil {
		return nil, err
	}
	return convertUser(row), nil
}

func (s *store) UpsertGoogleUser(ctx context.Context, params UpsertGoogleUserParams) (*User, error) {
	setter := &bobmodel.UserSetter{
		GoogleID:      omit.From(params.GoogleID),
		Email:         omit.From(params.Email),
		Name:          omitnull.FromPtr(params.Name),
		Picture:       omitnull.FromPtr(params.Picture),
		Locale:        omitnull.FromPtr(params.Locale),
		VerifiedEmail: omitnull.From(params.VerifiedEmail),
		LastLoginAt:   omitnull.From(params.LoginAt),
	}

	if cols := setter.SetColumns(); len(cols) == 0 {
		return nil, fmt.Errorf("userstore: empty column set for google user %s", params.GoogleID)
	}

	row, err := bobmodel.Users.Insert(setter).One(ctx, s.exec)
	if err != nil {
		return nil, err
	}

	return convertUser(row), nil
}

func convertUser(model *bobmodel.User) *User {
	return &User{
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

func assign(column bob.Expression, value bob.Expression) bob.Expression {
	return expr.Join{
		Sep:   " = ",
		Exprs: []bob.Expression{column, value},
	}
}
