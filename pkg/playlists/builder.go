package playlists

import (
	"context"
	"errors"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/zmb3/spotify"

	"github.com/jace-ys/spautofy/pkg/mail"
	"github.com/jace-ys/spautofy/pkg/users"
)

const (
	TimerangeShort  string = "short"
	TimerangeMedium string = "medium"
	TimerangeLong   string = "long"
)

var (
	ErrPlaylistNoSpotifyURL = errors.New("no spotify url found for playlist")
)

type BuilderFactory struct {
	baseURL       *url.URL
	mailer        mail.Mailer
	registry      *Registry
	users         *users.Registry
	authenticator *spotify.Authenticator
}

func NewBuilderFactory(baseURL *url.URL, mailer mail.Mailer, registry *Registry, users *users.Registry, authenticator *spotify.Authenticator) *BuilderFactory {
	return &BuilderFactory{
		baseURL:       baseURL,
		mailer:        mailer,
		registry:      registry,
		users:         users,
		authenticator: authenticator,
	}
}

type Builder struct {
	baseURL  *url.URL
	logger   log.Logger
	mailer   mail.Mailer
	registry *Registry
	client   *spotify.Client
	user     *users.User
}

func (bf *BuilderFactory) NewBuilder(ctx context.Context, logger log.Logger, userID string) (*Builder, error) {
	user, err := bf.users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	client, err := bf.ensureClient(ctx, user)
	if err != nil {
		return nil, err
	}

	return &Builder{
		baseURL:  bf.baseURL,
		logger:   log.With(logger, "user", userID),
		mailer:   bf.mailer,
		registry: bf.registry,
		client:   client,
		user:     user,
	}, nil
}

func (bf *BuilderFactory) ensureClient(ctx context.Context, user *users.User) (*spotify.Client, error) {
	client := bf.authenticator.NewClient(user.Token)

	if time.Now().Sub(user.Token.Expiry) > 0 {
		var err error
		user.Token, err = client.Token()
		if err != nil {
			return nil, err
		}

		_, err = bf.users.Update(ctx, user)
		if err != nil {
			return nil, err
		}

		client = bf.authenticator.NewClient(user.Token)
	}

	return &client, nil
}

func (b *Builder) Run(trackLimit int, withConfirm bool) func() {
	return func() {
		b.logger.Log("event", "playlist.build.started", "limit", trackLimit, "confirm", withConfirm)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		playlist, err := b.NewPlaylist(trackLimit, TimerangeShort)
		if err != nil {
			b.logger.Log("event", "playlist.new.failed", "error", err)
			return
		}

		id, err := b.registry.Create(ctx, playlist)
		if err != nil {
			b.logger.Log("event", "playlist.create.failed", "error", err)
			return
		}

		var playlistURL string
		if withConfirm {
			spautofyURL := *b.baseURL
			spautofyURL.Path = path.Join(spautofyURL.Path, "accounts", b.user.ID, "playlists", playlist.Name)
			playlistURL = spautofyURL.String()
		} else {
			err = b.Build(playlist)
			if err != nil {
				b.logger.Log("event", "playlist.build.failed", "error", err)
				return
			}

			id, err = b.registry.Update(ctx, playlist)
			if err != nil {
				b.logger.Log("event", "playlist.update.failed", "error", err)
				return
			}

			playlistURL = playlist.SpotifyURL
		}

		b.logger.Log("event", "playlist.build.finished", "id", id)

		err = b.mailer.SendNewPlaylistEmail(b.user, withConfirm, playlistURL)
		if err != nil {
			b.logger.Log("event", "email.send.failed", "error", err)
			return
		}

		b.logger.Log("event", "email.sent", "email", b.user.Email)
	}
}

func (b *Builder) NewPlaylist(limit int, timerange string) (*Playlist, error) {
	opts := &spotify.Options{
		Limit:     &limit,
		Timerange: &timerange,
	}

	tracks, err := b.client.CurrentUsersTopTracksOpt(opts)
	if err != nil {
		return nil, err
	}

	trackIDs := make([]spotify.ID, len(tracks.Tracks))
	for idx, track := range tracks.Tracks {
		trackIDs[idx] = track.ID
	}

	return &Playlist{
		UserID:      b.user.ID,
		Name:        time.Now().Format("Jan 2006"),
		Description: "A playlist put together for you by Spautofy based on your recent top tracks.",
		TrackIDs:    trackIDs,
	}, nil
}

func (b *Builder) Build(playlist *Playlist) error {
	spotifyPlaylist, err := b.client.CreatePlaylistForUser(playlist.UserID, playlist.Name, playlist.Description, false)
	if err != nil {
		return err
	}

	snapshotID, err := b.client.AddTracksToPlaylist(spotifyPlaylist.ID, playlist.TrackIDs...)
	if err != nil {
		return err
	}

	var ok bool
	playlist.SpotifyURL, ok = spotifyPlaylist.ExternalURLs["spotify"]
	if !ok {
		return ErrPlaylistNoSpotifyURL
	}

	playlist.ID = spotifyPlaylist.ID
	playlist.SnapshotID = snapshotID

	return nil
}

type Track struct {
	ID         spotify.ID
	Name       string
	Artists    string
	Album      string
	PreviewURL string
}

func (b *Builder) FetchTracks(trackIDs []spotify.ID) ([]*Track, error) {
	spotifyTracks, err := b.client.GetTracks(trackIDs...)
	if err != nil {
		return nil, err
	}

	tracks := make([]*Track, len(spotifyTracks))
	for i, track := range spotifyTracks {
		tracks[i] = &Track{
			ID:         track.ID,
			Name:       track.Name,
			Album:      track.Album.Name,
			PreviewURL: track.PreviewURL,
		}

		artists := make([]string, len(track.Artists))
		for j, artist := range track.Artists {
			artists[j] = artist.Name
		}

		tracks[i].Artists = strings.Join(artists, ", ")
	}

	return tracks, nil
}
