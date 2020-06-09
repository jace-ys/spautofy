package accounts

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jace-ys/go-library/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrAccountNotFound = errors.New("account not found")
	ErrAccountExists   = errors.New("account already exists")
)

type Account struct {
	UserID     string
	Schedule   string
	TrackLimit int
	WithEmail  bool
	CreatedAt  time.Time
}

func NewAccount(userID, schedule string, trackLimit int, withEmail bool) *Account {
	return &Account{
		UserID:     userID,
		Schedule:   schedule,
		TrackLimit: trackLimit,
		WithEmail:  withEmail,
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

func (r *Registry) Get(ctx context.Context, userID string) (*Account, error) {
	var account Account
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		SELECT user_id, schedule, track_limit, with_email, created_at
		FROM accounts
		WHERE user_id = $1
		`
		row := tx.QueryRowxContext(ctx, query, userID)
		return row.StructScan(&account)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrAccountNotFound
		default:
			return nil, err
		}
	}

	return &account, nil
}

func (r *Registry) Create(ctx context.Context, account *Account) (string, error) {
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		INSERT INTO accounts (user_id, schedule, track_limit, with_email)
		VALUES (:user_id, :schedule, :track_limit, :with_email)
		RETURNING user_id
		`
		stmt, err := tx.PrepareNamedContext(ctx, query)
		if err != nil {
			return err
		}
		row := stmt.QueryRowxContext(ctx, account)
		return row.Scan(&account.UserID)
	})
	if err != nil {
		var pqErr *pq.Error
		switch {
		case errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation":
			return "", ErrAccountExists
		default:
			return "", err
		}
	}

	return account.UserID, nil
}

func (r *Registry) CreateOrUpdate(ctx context.Context, account *Account) (string, error) {
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		INSERT INTO accounts (user_id, schedule, track_limit, with_email)
		VALUES (:user_id, :schedule, :track_limit, :with_email)
		ON CONFLICT (user_id)
		DO UPDATE SET
			schedule = EXCLUDED.schedule,
			track_limit = EXCLUDED.track_limit,
			with_email = EXCLUDED.with_email
		RETURNING user_id
		`
		stmt, err := tx.PrepareNamedContext(ctx, query)
		if err != nil {
			return err
		}
		row := stmt.QueryRowxContext(ctx, account)
		return row.Scan(&account.UserID)
	})
	if err != nil {
		var pqErr *pq.Error
		switch {
		case errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation":
			return "", ErrAccountExists
		default:
			return "", err
		}
	}

	return account.UserID, nil
}

func (r *Registry) Update(ctx context.Context, account *Account) (string, error) {
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		UPDATE accounts SET
			schedule = :schedule,
			track_limit = :track_limit,
			with_email = :with_email,
		WHERE user_id = :user_id
		RETURNING user_id
		`
		stmt, err := tx.PrepareNamedContext(ctx, query)
		if err != nil {
			return err
		}
		row := stmt.QueryRowxContext(ctx, account)
		return row.Scan(&account.UserID)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", ErrAccountNotFound
		default:
			return "", err
		}
	}

	return account.UserID, nil
}

func (r *Registry) Delete(ctx context.Context, userID string) error {
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		DELETE FROM accounts
		WHERE user_id = $1
		RETURNING user_id
		`
		row := tx.QueryRowContext(ctx, query, userID)
		return row.Scan(&userID)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrAccountNotFound
		default:
			return err
		}
	}

	return nil
}
