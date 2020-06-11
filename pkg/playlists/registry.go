package playlists

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jace-ys/go-library/postgres"
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
	SnapshotID  string
	CreatedAt   time.Time
}

type PlaylistEnvelope struct {
	*Playlist
	Tracks pq.StringArray
}

type Registry struct {
	database *postgres.Client
}

func NewRegistry(postgres *postgres.Client) *Registry {
	return &Registry{
		database: postgres,
	}
}

func (r *Registry) Get(ctx context.Context, userID, name string) (*Playlist, error) {
	var envelope PlaylistEnvelope
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		SELECT id, user_id, name, description, tracks, snapshot_id, created_at
		FROM playlists
		WHERE user_id = $1 AND name = $2
		`
		row := tx.QueryRowxContext(ctx, query, userID, name)
		return row.StructScan(&envelope)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrPlaylistNotFound
		default:
			return nil, err
		}
	}

	envelope.Playlist.TrackIDs = make([]spotify.ID, len(envelope.Tracks))
	for idx, track := range envelope.Tracks {
		envelope.Playlist.TrackIDs[idx] = spotify.ID(track)
	}

	return envelope.Playlist, nil
}

func (r *Registry) Create(ctx context.Context, playlist *Playlist) (spotify.ID, error) {
	envelope := &PlaylistEnvelope{
		Playlist: playlist,
		Tracks:   make(pq.StringArray, len(playlist.TrackIDs)),
	}

	for idx, track := range playlist.TrackIDs {
		envelope.Tracks[idx] = string(track)
	}

	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		INSERT INTO playlists (id, user_id, name, description, tracks, snapshot_id)
		VALUES (:id, :user_id, :name, :description, :tracks, :snapshot_id)
		RETURNING id
		`
		stmt, err := tx.PrepareNamedContext(ctx, query)
		if err != nil {
			return err
		}
		row := stmt.QueryRowxContext(ctx, envelope)
		return row.Scan(&playlist.ID)
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

	return playlist.ID, nil
}

func (r *Registry) Update(ctx context.Context, playlist *Playlist) (spotify.ID, error) {
	envelope := &PlaylistEnvelope{
		Playlist: playlist,
		Tracks:   make(pq.StringArray, len(playlist.TrackIDs)),
	}

	for idx, track := range playlist.TrackIDs {
		envelope.Tracks[idx] = string(track)
	}

	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		UPDATE playlists SET
			id = :id,
			description = :description,
			tracks = :tracks,
			snapshot_id = :snapshot_id
		WHERE user_id = :user_id AND name = :name
		RETURNING id
		`
		stmt, err := tx.PrepareNamedContext(ctx, query)
		if err != nil {
			return err
		}
		row := stmt.QueryRowxContext(ctx, envelope)
		return row.Scan(&playlist.ID)
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return "", ErrPlaylistNotFound
		default:
			return "", err
		}
	}

	return playlist.ID, nil
}

func (r *Registry) Delete(ctx context.Context, userID, name string) error {
	var id string
	err := r.database.Transact(ctx, func(tx *sqlx.Tx) error {
		query := `
		DELETE FROM playlists
		WHERE user_id = $1 AND name = $2
		RETURNING id
		`
		row := tx.QueryRowContext(ctx, query, userID, name)
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
