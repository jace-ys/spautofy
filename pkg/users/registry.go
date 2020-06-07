package users

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jace-ys/go-library/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

type User struct {
	*spotify.PrivateUser
	*oauth2.Token
	CreatedAt time.Time
}

func NewUser(spotifyUser *spotify.PrivateUser, token *oauth2.Token) *User {
	return &User{
		PrivateUser: spotifyUser,
		Token:       token,
	}
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
	var user User
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		SELECT u.id, u.email, u.display_name, u.created_at
		FROM users AS u
		WHERE u.id = $1
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

	return &user, nil
}

func (r *Registry) Create(ctx context.Context, user *User) (string, error) {
	var id string
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		INSERT INTO users (id, email, display_name, access_token, token_type, refresh_token, expiry)
		VALUES (:id, :email, :display_name, :access_token, :token_type, :refresh_token, :expiry)
		ON CONFLICT (id)
		DO UPDATE SET
			email = EXCLUDED.email,
			display_name = EXCLUDED.display_name,
			access_token = EXCLUDED.access_token,
			token_type = EXCLUDED.token_type,
			refresh_token = EXCLUDED.refresh_token,
			expiry = EXCLUDED.expiry
		RETURNING id
		`
		stmt, err := tx.PrepareNamedContext(ctx, query)
		if err != nil {
			return err
		}
		row := stmt.QueryRowxContext(ctx, user)
		return row.Scan(&id)
	})
	if err != nil {
		var pqErr *pq.Error
		switch {
		case errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation":
			return "", ErrUserExists
		default:
			return "", err
		}
	}

	return id, nil
}

func (r *Registry) Delete(ctx context.Context, userID string) error {
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		DELETE FROM users
		WHERE id = $1
		RETURNING id
		`
		row := tx.QueryRowContext(ctx, query, userID)
		return row.Scan(&userID)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrUserNotFound
		default:
			return err
		}
	}

	return nil
}
