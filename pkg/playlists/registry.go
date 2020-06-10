package playlists

import (
	"context"
	"database/sql"
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
	SnapshotID  string
	CreatedAt   time.Time
}

type PlaylistEnvelope struct {
	*Playlist
	Tracks pq.StringArray
}

func NewPlaylist(userID string, trackIDs []spotify.ID) *Playlist {
	return &Playlist{
		UserID:      userID,
		Name:        time.Now().Format("Jan 2006"),
		Description: "A playlist put together for you by Spautofy based on your recent top tracks.",
		TrackIDs:    trackIDs,
	}
}

func (b *Builder) Get(ctx context.Context, userID, name string) (*Playlist, error) {
	var envelope PlaylistEnvelope
	err := b.database.Transact(ctx, func(tx *sqlx.Tx) error {
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

func (b *Builder) Create(ctx context.Context, playlist *Playlist) (string, error) {
	envelope := &PlaylistEnvelope{
		Playlist: playlist,
		Tracks:   make(pq.StringArray, len(playlist.TrackIDs)),
	}

	for idx, track := range playlist.TrackIDs {
		envelope.Tracks[idx] = string(track)
	}

	err := b.database.Transact(ctx, func(tx *sqlx.Tx) error {
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

	return string(playlist.ID), nil
}

func (b *Builder) Delete(ctx context.Context, userID, name string) error {
	var id string
	err := b.database.Transact(ctx, func(tx *sqlx.Tx) error {
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
