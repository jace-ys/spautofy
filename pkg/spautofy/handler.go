package spautofy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/jace-ys/go-library/postgres"
	"github.com/robfig/cron/v3"
	"github.com/zmb3/spotify"

	"github.com/jace-ys/spautofy/pkg/user"
)

type Config struct {
	ClientID     string
	Secret       string
	RedirectHost string
}

type Handler struct {
	logger        log.Logger
	server        *http.Server
	runner        *cron.Cron
	users         *user.Registry
	authenticator *spotify.Authenticator
}

func NewHandler(logger log.Logger, cfg *Config, postgres *postgres.Client) *Handler {
	redirectURL := fmt.Sprintf("http://%s/login/callback", cfg.RedirectHost)
	authenticator := spotify.NewAuthenticator(redirectURL, scopes...)
	authenticator.SetAuthInfo(cfg.ClientID, cfg.Secret)

	handler := &Handler{
		logger:        logger,
		server:        &http.Server{},
		runner:        cron.New(),
		users:         user.NewRegistry(postgres),
		authenticator: &authenticator,
	}
	handler.server.Handler = handler.router()

	return handler
}

func (h *Handler) router() http.Handler {
	router := mux.NewRouter()

	router.HandleFunc("/", h.renderIndex).Methods(http.MethodGet)

	login := router.PathPrefix("/login").Subrouter()
	login.HandleFunc("", h.loginRedirect).Methods(http.MethodGet)
	login.HandleFunc("/callback", h.loginCallback).Methods(http.MethodGet)

	protected := router.PathPrefix("/account").Subrouter()
	protected.Use(h.authMiddleware)
	protected.HandleFunc("/{id:[0-9]+}/manage", h.renderAccount).Methods(http.MethodGet)

	router.NotFoundHandler = http.HandlerFunc(h.render404)

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

func (h *Handler) StartRunner() error {
	h.logger.Log("event", "runner.started")
	defer h.logger.Log("event", "runner.stopped")

	h.runner.Run()
	return nil
}

func (h *Handler) Shutdown(ctx context.Context) error {
	if err := h.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}
	return nil
}

type httpError struct {
	Error struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}

func (h *Handler) sendJSON(w http.ResponseWriter, status int, v interface{}) {
	if err, ok := v.(error); ok {
		var httpErr httpError
		httpErr.Error.Status = status
		httpErr.Error.Message = err.Error()
		v = httpErr
	}

	response, err := json.Marshal(v)
	if err != nil {
		h.logger.Log("event", "response.encoded", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Spautofy is currently unavailable. Please try again later."))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(response))
}
