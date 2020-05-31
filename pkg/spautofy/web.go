package spautofy

import (
	"html/template"
	"net/http"
)

var tmpls *template.Template

func init() {
	tmpls = template.Must(template.ParseGlob("web/templates/*.html"))
}

func (h *Handler) renderIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.logger.Log("event", "template.rendered", "template", "index")
		tmpls.ExecuteTemplate(w, "index", nil)
	}
}

func (h *Handler) renderAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(userIDKey{}).(string)
		h.logger.Log("event", "template.rendered", "template", "account", "user", userID)
		tmpls.ExecuteTemplate(w, "account", nil)
	}
}

func (h *Handler) renderError(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.logger.Log("event", "template.rendered", "template", "error", "path", r.URL.Path, "status", status)

		var message string
		switch status {
		case http.StatusNotFound:
			message = "404 Not Found"
		default:
			message = "Spautofy is currently unavailable. Please try again later."
		}

		tmpls.ExecuteTemplate(w, "error", message)
	}
}
