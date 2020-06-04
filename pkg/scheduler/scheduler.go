package scheduler

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-kit/kit/log"
	"github.com/jace-ys/go-library/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/robfig/cron/v3"
)

var (
	ErrScheduleNotFound = errors.New("schedule not found")
	ErrScheduleExists   = errors.New("schedule already exists")
)

type Schedule struct {
	ID        cron.EntryID
	UserID    string
	Spec      string
	WithEmail bool
}

type Scheduler struct {
	logger   log.Logger
	runner   *cron.Cron
	database *postgres.Client
}

func NewScheduler(logger log.Logger, postgres *postgres.Client) *Scheduler {
	return &Scheduler{
		logger:   logger,
		runner:   cron.New(),
		database: postgres,
	}
}

func (s *Scheduler) List(ctx context.Context) ([]*Schedule, error) {
	var schedules []*Schedule
	err := s.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		SELECT s.id, s.user_id, s.spec, s.with_email
		FROM schedules AS s
		`
		rows, err := tx.QueryxContext(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var schedule Schedule
			if err := rows.StructScan(&schedule); err != nil {
				return err
			}
			schedules = append(schedules, &schedule)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return schedules, nil
}

func (s *Scheduler) Get(ctx context.Context, userID string) (*Schedule, error) {
	var schedule Schedule
	err := s.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		SELECT s.id, s.user_id, s.spec, s.email
		FROM schedules AS s
		WHERE s.user_id = $1
		`
		row := tx.QueryRowxContext(ctx, query, userID)
		return row.StructScan(&schedule)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrScheduleNotFound
		default:
			return nil, err
		}
	}

	return &schedule, nil
}

func (s *Scheduler) Create(ctx context.Context, userID string, spec string, withEmail bool) (cron.EntryID, error) {
	_, err := cron.ParseStandard(spec)
	if err != nil {
		return 0, err
	}

	id, err := s.runner.AddFunc(spec, func() { s.logger.Log("event", "task.started", "user", userID) })
	if err != nil {
		return 0, err
	}

	schedule := &Schedule{
		ID:        id,
		UserID:    userID,
		Spec:      spec,
		WithEmail: withEmail,
	}

	err = s.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		INSERT INTO schedules (id, user_id, spec, with_email)
		VALUES (:id, :user_id, :spec, :with_email)
		ON CONFLICT (user_id)
		DO UPDATE SET id = EXCLUDED.id, spec = EXCLUDED.spec, with_email = EXCLUDED.with_email
		RETURNING id
		`
		stmt, err := tx.PrepareNamedContext(ctx, query)
		if err != nil {
			return err
		}
		row := stmt.QueryRowxContext(ctx, schedule)
		return row.Scan(&id)
	})
	if err != nil {
		var pqErr *pq.Error
		switch {
		case errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation":
			return 0, ErrScheduleExists
		default:
			return 0, err
		}
	}

	return id, nil
}

func (s *Scheduler) Delete(ctx context.Context, userID string) error {
	var id cron.EntryID
	err := s.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		DELETE FROM schedules
		WHERE user_id = $1
		RETURNING id
		`
		row := tx.QueryRowContext(ctx, query, userID)
		return row.Scan(&id)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrScheduleNotFound
		default:
			return err
		}
	}

	s.runner.Remove(id)
	return nil
}
