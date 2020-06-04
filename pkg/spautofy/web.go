package spautofy

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

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
		user, err := h.users.Get(r.Context(), mux.Vars(r)["userID"])
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

		h.logger.Log("event", "template.rendered", "template", "account", "user", user.ID)
		tmpls.ExecuteTemplate(w, "account", user.PrivateUser)
	}
}

func (h *Handler) updateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			h.logger.Log("event", "form.parse.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		userID := mux.Vars(r)["userID"]

		frequency, err := strconv.Atoi(r.PostForm.Get("frequency"))
		if err != nil {
			h.logger.Log("event", "form.parse.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}
		spec := fmt.Sprintf("0 0 1 1/%d *", 12/frequency)

		var withEmail bool
		if r.PostForm.Get("email") != "" {
			withEmail = true
		}

		err = h.scheduler.Delete(r.Context(), userID)
		if err != nil {
			switch {
			case errors.Is(err, scheduler.ErrScheduleNotFound):
				// no-op
			default:
				h.logger.Log("event", "schedule.delete.failed", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}
		}

		scheduleID, err := h.scheduler.Create(r.Context(), userID, spec, withEmail)
		if err != nil {
			h.logger.Log("event", "schedule.create.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/account/%s", userID))
		w.WriteHeader(http.StatusFound)

		h.logger.Log("event", "account.updated", "user", userID, "schedule", scheduleID)
	}
}

func (h *Handler) renderError(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type renderData struct {
			Authenticated bool
			UserID        string
			Message       string
		}

		var data renderData
		userID, ok := r.Context().Value(userIDKey{}).(string)
		if !ok {
			data.Authenticated = false
		} else {
			data.Authenticated = true
			data.UserID = userID
		}

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
