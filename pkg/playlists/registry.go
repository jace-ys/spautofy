package playlists

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zmb3/spotify"
)

var (
	ErrPlaylistNotFound = errors.New("playlist not found")
	ErrPlaylistExists   = errors.New("playlist already exists")
)

type Playlist struct {
	ID          spotify.ID
	UserID      string
	Name        string
	Description string
	TrackIDs    []spotify.ID
	Endpoint    string
	SnapshotID  string
	CreatedAt   time.Time
}

func NewPlaylist(userID string, trackIDs []spotify.ID) *Playlist {
	return &Playlist{
		UserID:      userID,
		Name:        time.Now().Format("Jan 2006"),
		Description: "A playlist put together for you by Spautofy based on your recent top tracks.",
		TrackIDs:    trackIDs,
	}
}

func (b *Builder) Get(ctx context.Context, name string) (*Playlist, error) {
	var playlist Playlist
	err := b.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		SELECT id, user_id, name, description, tracks, endpoint, snapshot_id, created_at
		FROM playlists
		WHERE name = $1
		`
		row := tx.QueryRowxContext(ctx, query, name)
		return row.StructScan(&playlist)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrPlaylistNotFound
		default:
			return nil, err
		}
	}

	return &playlist, nil
}

func (b *Builder) Create(ctx context.Context, playlist *Playlist) (string, error) {
	dbPlaylist := struct {
		*Playlist
		Tracks interface {
			driver.Valuer
			sql.Scanner
		}
	}{
		Playlist: playlist,
		Tracks:   pq.Array(playlist.TrackIDs),
	}

	var id string
	err := b.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		INSERT INTO playlists
			(id, user_id, name, description, tracks, endpoint, snapshot_id)
		VALUES
			(:id, :user_id, :name, :description, :tracks, :endpoint, :snapshot_id)
		RETURNING id
		`
		stmt, err := tx.PrepareNamedContext(ctx, query)
		if err != nil {
			return err
		}
		row := stmt.QueryRowxContext(ctx, dbPlaylist)
		return row.Scan(&id)
	})
	if err != nil {
		var pqErr *pq.Error
		switch {
		case errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation":
			return "", ErrPlaylistExists
		default:
			return "", err
		}
	}

	return id, nil
}

func (b *Builder) Delete(ctx context.Context, name string) error {
	var id string
	err := b.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		DELETE FROM playlists
		WHERE name = $1
		RETURNING id
		`
		row := tx.QueryRowContext(ctx, query, name)
		return row.Scan(&id)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrPlaylistNotFound
		default:
			return err
		}
	}

	return nil
}
