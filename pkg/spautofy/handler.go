package spautofy

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/jace-ys/go-library/postgres"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jace-ys/spautofy/pkg/user"
)

type Handler struct {
	logger log.Logger
	server *http.Server
	users  *user.Registry
}

func NewHandler(logger log.Logger, postgres *postgres.Client) *Handler {
	handler := &Handler{
		logger: logger,
		server: &http.Server{},
		users:  user.NewRegistry(postgres),
	}
	handler.server.Handler = handler.router()
	return handler
}

func (h *Handler) router() http.Handler {
	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.Handle("/health", promhttp.Handler()).Methods(http.MethodGet)

	return router
}

func (h *Handler) StartServer(port int) error {
	h.logger.Log("event", "server.started", "port", port)
	defer h.logger.Log("event", "server.stopped")
	h.server.Addr = fmt.Sprintf(":%d", port)
	if err := h.server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

func (h *Handler) Shutdown(ctx context.Context) error {
	if err := h.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}
	return nil
}
