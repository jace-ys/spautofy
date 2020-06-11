package spautofy

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/zmb3/spotify"

	"github.com/jace-ys/spautofy/pkg/accounts"
	"github.com/jace-ys/spautofy/pkg/playlists"
	"github.com/jace-ys/spautofy/pkg/scheduler"
	"github.com/jace-ys/spautofy/pkg/users"
	"github.com/jace-ys/spautofy/pkg/web/templates"
)

var tmpls *template.Template

func init() {
	assets := make([]string, len(templates.AssetNames()))
	for idx, name := range templates.AssetNames() {
		assets[idx] = string(templates.MustAsset(name))
	}

	tmpls = template.Must(template.New("tmpls").Parse(strings.Join(assets, "")))
}

func (h *Handler) renderIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.logger.Log("event", "template.rendered", "template", "index")
		tmpls.ExecuteTemplate(w, "index", nil)
	}
}

func (h *Handler) renderAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			User       *spotify.PrivateUser
			Frequency  int
			TrackLimit int
			WithEmail  bool
			Next       time.Time
		}{
			WithEmail:  true,
			TrackLimit: 20,
			Frequency:  12,
		}

		userID := mux.Vars(r)["userID"]

		user, err := h.users.Get(r.Context(), userID)
		if err != nil {
			switch {
			case errors.Is(err, users.ErrUserNotFound):
				h.renderError(http.StatusNotFound).ServeHTTP(w, r)
				return
			default:
				h.logger.Log("event", "user.get.failed", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}
		}
		data.User = user.PrivateUser

		account, err := h.accounts.Get(r.Context(), user.ID)
		if err != nil {
			switch {
			case errors.Is(err, accounts.ErrAccountNotFound):
				// no-op
			default:
				h.logger.Log("event", "account.get.failed", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}
		} else {
			data.Frequency = scheduler.SpecToFrequency(account.Schedule)
			data.TrackLimit = account.TrackLimit
			data.WithEmail = account.WithEmail
			data.Next = scheduler.GetNext(account.Schedule)
		}

		h.logger.Log("event", "template.rendered", "template", "account", "user", user.ID)
		tmpls.ExecuteTemplate(w, "account", data)
	}
}

func (h *Handler) renderPlaylist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			UserID string
			Name   string
			Link   string
			Tracks []*playlists.Track
		}{}

		userID := mux.Vars(r)["userID"]
		playlistName := mux.Vars(r)["playlistName"]

		playlist, err := h.playlists.Get(r.Context(), userID, playlistName)
		if err != nil {
			switch {
			case errors.Is(err, playlists.ErrPlaylistNotFound):
				h.renderError(http.StatusNotFound).ServeHTTP(w, r)
				return
			default:
				h.logger.Log("event", "playlist.get.failed", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}
		}

		builder, err := h.builder.NewBuilder(r.Context(), playlist.UserID)
		if err != nil {
			h.logger.Log("event", "builder.new.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		data.UserID = playlist.UserID
		data.Name = playlist.Name
		data.Tracks, err = builder.FetchTracks(playlist.TrackIDs)
		if err != nil {
			h.logger.Log("event", "tracks.fetch.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		if playlist.SnapshotID != "" {
			data.Link = fmt.Sprintf("https://open.spotify.com/playlist/%s", playlist.ID)
		}

		h.logger.Log("event", "template.rendered", "template", "playlist", "user", userID, "playlist", playlist.Name)
		tmpls.ExecuteTemplate(w, "playlist", data)
	}
}

func (h *Handler) renderError(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Message string
		}{}

		switch status {
		case http.StatusUnauthorized:
			data.Message = "You need to be logged in to view this page."
		case http.StatusForbidden:
			data.Message = "You do not have permissions to view this page."
		case http.StatusNotFound:
			data.Message = "The requested content could not be found."
		default:
			data.Message = "Spautofy is currently unavailable. Please try again later."
		}

		h.logger.Log("event", "template.rendered", "template", "error", "path", r.URL.Path, "status", status)
		tmpls.ExecuteTemplate(w, "error", data)
	}
}
