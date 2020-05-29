package spautofy

import (
	"fmt"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/zmb3/spotify"
)

const (
	// TODO: generate this state programmatically and cache it for a short period
	authState   = "spautofy"
	redirectURL = "http://localhost:8080/api/v1/login/callback"
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

	spew.Dump(user)

	w.Header().Set("Location", fmt.Sprintf("/api/v1/health?token=%s", token.AccessToken))
	w.WriteHeader(http.StatusFound)
}
