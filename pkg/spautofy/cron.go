package spautofy

import (
	"net/http"
	"time"

	"github.com/robfig/cron/v3"
)

func (h *Handler) listSchedules() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schedules := make([]interface{}, len(h.runner.Entries()))
		for i, e := range h.runner.Entries() {
			schedules[i] = struct {
				ID   cron.EntryID `json:"id"`
				Next time.Time    `json:"next"`
				Prev time.Time    `json:"prev"`
			}{
				ID:   e.ID,
				Next: e.Next,
				Prev: e.Prev,
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}
