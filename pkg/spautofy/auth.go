package spautofy

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/zmb3/spotify"

	"github.com/jace-ys/spautofy/pkg/users"
)

var (
	scopes = []string{
		spotify.ScopeUserReadEmail,
	}
)

func (h *Handler) loginRedirect() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.logger.Log("event", "login.started")

		session, err := h.sessions.Create(w, r)
		if err != nil {
			h.logger.Log("event", "login.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		http.Redirect(w, r, h.authenticator.AuthURL(session.GetID()), http.StatusFound)
	}
}

func (h *Handler) loginCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := h.sessions.Get(r)
		if err != nil {
			h.logger.Log("event", "login.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		token, err := h.authenticator.Token(session.GetID(), r)
		if err != nil {
			h.logger.Log("event", "login.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		client := h.authenticator.NewClient(token)
		spotifyUser, err := client.CurrentUser()
		if err != nil {
			h.logger.Log("event", "login.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		// TODO: update user if exists
		user := users.NewUser(spotifyUser, token)
		userID, err := h.users.Create(r.Context(), user)
		if err != nil {
			switch {
			case errors.Is(err, users.ErrUserExists):
				userID = user.ID
			default:
				h.logger.Log("event", "user.created", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}
		} else {
			h.logger.Log("event", "user.created", "user", userID)
		}

		values := map[interface{}]interface{}{
			"userID": userID,
		}

		session, err = h.sessions.Update(w, r, values)
		if err != nil {
			h.logger.Log("event", "login.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/account/%s", spotifyUser.ID))
		w.WriteHeader(http.StatusFound)

		h.logger.Log("event", "login.finished", "session", session.GetID(), "user", userID)
	}
}

func (h *Handler) logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.logger.Log("event", "logout.started")

		session, err := h.sessions.Delete(w, r)
		if err != nil {
			h.logger.Log("event", "logout.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)

		h.logger.Log("event", "logout.finished", "session", session.GetID(), "user", session.Values["userID"])
	}
}

type userIDKey struct{}

func (h *Handler) middlewareAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := h.sessions.Get(r)
		if err != nil {
			h.logger.Log("event", "authenticate.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		userID, ok := session.Values["userID"]
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
