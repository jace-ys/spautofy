package spautofy

import (
	"html/template"
	"net/http"
)

var tmpls *template.Template

func init() {
	tmpls = template.Must(template.ParseGlob("web/templates/*.html"))
}

func (h *Handler) renderIndex(w http.ResponseWriter, r *http.Request) {
	h.logger.Log("event", "template.rendered", "template", "index")
	tmpls.ExecuteTemplate(w, "index", nil)
}

func (h *Handler) renderAccount(w http.ResponseWriter, r *http.Request) {
	h.logger.Log("event", "template.rendered", "template", "account")
	tmpls.ExecuteTemplate(w, "account", nil)
}

func (h *Handler) render404(w http.ResponseWriter, r *http.Request) {
	h.logger.Log("event", "template.rendered", "template", "404", "path", r.URL.Path)
	tmpls.ExecuteTemplate(w, "404", nil)
}
