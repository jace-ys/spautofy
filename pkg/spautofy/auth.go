package spautofy

import (
	"fmt"
	"net/http"

	"github.com/zmb3/spotify"
)

const (
	// TODO: generate this state programmatically and cache it for a short period
	authState = "spautofy"
)

var (
	scopes = []string{
		spotify.ScopeUserReadEmail,
	}
)

func (h *Handler) loginRedirect(w http.ResponseWriter, r *http.Request) {
	h.logger.Log("event", "login.started")
	http.Redirect(w, r, h.authenticator.AuthURL(authState), http.StatusFound)
}

func (h *Handler) loginCallback(w http.ResponseWriter, r *http.Request) {
	defer h.logger.Log("event", "login.finished")

	token, err := h.authenticator.Token(authState, r)
	if err != nil {
		h.sendJSON(w, http.StatusForbidden, err)
		return
	}

	if state := r.FormValue("state"); state != authState {
		h.sendJSON(w, http.StatusForbidden, nil)
		return
	}

	client := h.authenticator.NewClient(token)

	user, err := client.CurrentUser()
	if err != nil {
		h.sendJSON(w, http.StatusNotFound, err)
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/account/%s/manage", user.ID))
	w.WriteHeader(http.StatusFound)
}

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
