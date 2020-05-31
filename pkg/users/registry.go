package users

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jace-ys/go-library/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/zmb3/spotify"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type User struct {
	*spotify.PrivateUser
}

type Registry struct {
	database *postgres.Client
}

func NewRegistry(postgres *postgres.Client) *Registry {
	return &Registry{
		database: postgres,
	}
}

func (r *Registry) Get(ctx context.Context, id string) (*User, error) {
	var user spotify.PrivateUser
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		SELECT s.id
		FROM users AS u
		WHERE u.id=$1
		`
		row := tx.QueryRowxContext(ctx, query, id)
		return row.StructScan(&user)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrUserNotFound
		default:
			return nil, err
		}
	}
	return &User{&user}, nil
}
