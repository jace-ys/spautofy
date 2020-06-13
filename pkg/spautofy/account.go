package spautofy

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/jace-ys/spautofy/pkg/accounts"
	"github.com/jace-ys/spautofy/pkg/scheduler"
)

func (h *Handler) updateAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account, err := h.parseAccountForm(r)
		if err != nil {
			h.logger.Log("event", "form.parse.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		userID, err := h.accounts.CreateOrUpdate(r.Context(), account)
		if err != nil {
			h.logger.Log("event", "account.upsert.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
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

		builder, err := h.builder.NewBuilder(r.Context(), h.logger, userID)
		if err != nil {
			h.logger.Log("event", "builder.new.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		cmd := builder.Run(account.TrackLimit, account.WithConfirm)

		schedule := scheduler.NewSchedule(userID, account.Schedule, cmd)
		scheduleID, err := h.scheduler.Create(r.Context(), schedule)
		if err != nil {
			h.logger.Log("event", "schedule.create.failed", "error", err)
			h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/accounts/%s", userID))
		w.WriteHeader(http.StatusFound)

		h.logger.Log("event", "account.updated", "user", userID, "schedule", scheduleID)
	}
}

func (h *Handler) parseAccountForm(r *http.Request) (*accounts.Account, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	frequency, err := strconv.Atoi(r.PostForm.Get("frequency"))
	if err != nil {
		return nil, err
	}

	limit, err := strconv.Atoi(r.PostForm.Get("limit"))
	if err != nil {
		return nil, err
	}

	_, withConfirm := r.PostForm["confirm"]

	account := accounts.NewAccount(mux.Vars(r)["userID"], scheduler.FrequencyToSpec(frequency), limit, withConfirm)
	return account, nil
}

func (h *Handler) deleteAccount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["userID"]

		err := h.scheduler.Delete(r.Context(), userID)
		if err != nil {
			switch {
			case errors.Is(err, scheduler.ErrScheduleNotFound):
				h.renderError(http.StatusNotFound).ServeHTTP(w, r)
				return
			default:
				h.logger.Log("event", "schedule.delete.failed", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}
		}

		err = h.accounts.Delete(r.Context(), userID)
		if err != nil {
			switch {
			case errors.Is(err, accounts.ErrAccountNotFound):
				h.renderError(http.StatusNotFound).ServeHTTP(w, r)
				return
			default:
				h.logger.Log("event", "account.delete.failed", "error", err)
				h.renderError(http.StatusInternalServerError).ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("Location", fmt.Sprintf("/accounts/%s", userID))
		w.WriteHeader(http.StatusFound)

		h.logger.Log("event", "account.deleted", "user", userID)
	}
}
