package spautofy

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/zmb3/spotify"

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
			User      *spotify.PrivateUser
			WithEmail bool
			Frequency int
		}{
			WithEmail: true,
			Frequency: 12,
		}

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
		data.User = user.PrivateUser

		schedule, err := h.scheduler.Get(r.Context(), user.ID)
		if err != nil {
			switch {
			case errors.Is(err, scheduler.ErrScheduleNotFound):
				// no-op
			default:
				h.logger.Log("event", "schedule.get.failed", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}
		} else {
			data.Frequency = scheduler.SpecToFrequency(schedule.Spec)
			data.WithEmail = schedule.WithEmail
		}

		h.logger.Log("event", "template.rendered", "template", "account", "user", user.ID)
		tmpls.ExecuteTemplate(w, "account", data)
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

		frequency, err := strconv.Atoi(r.PostForm.Get("frequency"))
		if err != nil {
			h.logger.Log("event", "form.parse.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		userID := mux.Vars(r)["userID"]
		spec := scheduler.FrequencyToSpec(frequency)
		_, withEmail := r.PostForm["email"]

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
