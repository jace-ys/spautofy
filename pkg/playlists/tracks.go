package playlists

import (
	"context"
	"strings"

	"github.com/zmb3/spotify"
)

type Track struct {
	ID         spotify.ID
	Name       string
	Artists    string
	Album      string
	PreviewURL string
}

func (b *Builder) FetchTracks(ctx context.Context, userID string, trackIDs []spotify.ID) ([]*Track, error) {
	user, err := b.users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	err = b.ensureClient(ctx, user)
	if err != nil {
		return nil, err
	}

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
