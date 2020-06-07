package playlists

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/jace-ys/go-library/postgres"
	"github.com/zmb3/spotify"

	"github.com/jace-ys/spautofy/pkg/users"
)

const (
	TimerangeShort  string = "short"
	TimerangeMedium string = "medium"
	TimerangeLong   string = "long"
)

type Playlist struct {
	ID          spotify.ID
	UserID      string
	Name        string
	Description string
	Tracks      []spotify.ID
	Endpoint    string
	SnapshotID  string
	CreatedAt   time.Time
}

func NewPlaylist(userID string, tracks []spotify.ID) *Playlist {
	return &Playlist{
		UserID:      userID,
		Name:        time.Now().Format("Jan 2006"),
		Description: "A playlist put together for you by Spautofy based on your recent top tracks.",
		Tracks:      tracks,
	}
}

type Builder struct {
	logger        log.Logger
	database      *postgres.Client
	users         *users.Registry
	authenticator *spotify.Authenticator
	client        *spotify.Client
}

func NewBuilder(logger log.Logger, postgres *postgres.Client, users *users.Registry, authenticator *spotify.Authenticator) *Builder {
	return &Builder{
		logger:        logger,
		database:      postgres,
		users:         users,
		authenticator: authenticator,
	}
}

func (b *Builder) Run(userID string, limit int, withEmail bool) func() {
	return func() {
		logger := log.With(b.logger, "user", userID, "email", withEmail)
		logger.Log("event", "playlist.create.started")
		defer logger.Log("event", "playlist.create.finished")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		user, err := b.users.Get(ctx, userID)
		if err != nil {
			logger.Log("event", "user.get.failed", "error", err)
			return
		}

		err = b.ensureClient(ctx, user)
		if err != nil {
			logger.Log("event", "client.ensure.failed", "error", err)
			return
		}

		playlist, err := b.BuildPlaylist(user, limit, TimerangeShort, withEmail)
		if err != nil {
			logger.Log("event", "playlist.build.failed", "error", err)
			return
		}

		fmt.Println(playlist)
	}
}

func (b *Builder) ensureClient(ctx context.Context, user *users.User) error {
	client := b.authenticator.NewClient(user.Token)

	if time.Now().Sub(user.Token.Expiry) > 0 {
		var err error
		user.Token, err = client.Token()
		if err != nil {
			return err
		}

		_, err = b.users.Update(ctx, user)
		if err != nil {
			return err
		}

		client = b.authenticator.NewClient(user.Token)
	}

	b.client = &client
	return nil
}

func (b *Builder) BuildPlaylist(user *users.User, limit int, timerange string, withEmail bool) (*Playlist, error) {
	opts := &spotify.Options{
		Limit:     &limit,
		Timerange: &timerange,
	}

	tracks, err := b.getTrackIDs(opts)
	if err != nil {
		return nil, err
	}

	playlist := NewPlaylist(user.ID, tracks)

	if withEmail {
		// TODO: send email with playlist data
		return playlist, nil
	}

	return playlist, b.buildPlaylist(playlist)
}

func (b *Builder) getTrackIDs(opts *spotify.Options) ([]spotify.ID, error) {
	tracks, err := b.client.CurrentUsersTopTracksOpt(opts)
	if err != nil {
		return nil, err
	}

	trackIDs := make([]spotify.ID, len(tracks.Tracks))
	for idx, track := range tracks.Tracks {
		trackIDs[idx] = track.ID
	}

	return trackIDs, nil
}

func (b *Builder) buildPlaylist(playlist *Playlist) error {
	spotifyPlaylist, err := b.client.CreatePlaylistForUser(playlist.UserID, playlist.Name, playlist.Description, false)
	if err != nil {
		return err
	}

	snapshotID, err := b.client.AddTracksToPlaylist(spotifyPlaylist.ID, playlist.Tracks...)
	if err != nil {
		return err
	}

	playlist.ID = spotifyPlaylist.ID
	playlist.Endpoint = spotifyPlaylist.Endpoint
	playlist.SnapshotID = snapshotID

	return nil
}
