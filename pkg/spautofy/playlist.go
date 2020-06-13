package spautofy

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zmb3/spotify"

	"github.com/jace-ys/spautofy/pkg/playlists"
)

func (h *Handler) createPlaylist() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tracks, err := h.parsePlaylistForm(r)
		if err != nil {
			h.logger.Log("event", "form.parse.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

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
		playlist.TrackIDs = tracks

		if playlist.SnapshotID == "" {
			builder, err := h.builder.NewBuilder(r.Context(), h.logger, playlist.UserID)
			if err != nil {
				h.logger.Log("event", "builder.new.failed", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}

			err = builder.Build(playlist)
			if err != nil {
				h.logger.Log("event", "playlist.build.failed", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}

			id, err := h.playlists.Update(r.Context(), playlist)
			if err != nil {
				switch {
				case errors.Is(err, playlists.ErrPlaylistNotFound):
					h.renderError(http.StatusNotFound).ServeHTTP(w, r)
					return
				default:
					h.logger.Log("event", "playlist.update.failed", "error", err)
					h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
					return
				}
			}

			h.logger.Log("event", "playlist.built", "user", playlist.UserID, "id", id)
		}

		w.Header().Set("Location", fmt.Sprintf("/accounts/%s/playlists/%s", playlist.UserID, playlist.Name))
		w.WriteHeader(http.StatusFound)
	}
}

func (h *Handler) parsePlaylistForm(r *http.Request) ([]spotify.ID, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	var trackIDs []spotify.ID
	for k := range r.PostForm {
		trackIDs = append(trackIDs, spotify.ID(k))
	}

	return trackIDs, nil
}
