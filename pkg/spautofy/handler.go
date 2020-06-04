package spautofy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/jace-ys/go-library/postgres"
	"github.com/zmb3/spotify"

	"github.com/jace-ys/spautofy/pkg/scheduler"
	"github.com/jace-ys/spautofy/pkg/sessions"
	"github.com/jace-ys/spautofy/pkg/users"
	"github.com/jace-ys/spautofy/pkg/web/static"
)

type Config struct {
	ClientID     string
	Secret       string
	RedirectHost string
	SessionKey   string
}

type Handler struct {
	logger        log.Logger
	server        *http.Server
	scheduler     *scheduler.Scheduler
	users         *users.Registry
	authenticator *spotify.Authenticator
	sessions      *sessions.Manager
}

func NewHandler(logger log.Logger, cfg *Config, postgres *postgres.Client) *Handler {
	redirectURL := fmt.Sprintf("http://%s/login/callback", cfg.RedirectHost)
	authenticator := spotify.NewAuthenticator(redirectURL, scopes...)
	authenticator.SetAuthInfo(cfg.ClientID, cfg.Secret)

	handler := &Handler{
		logger:        logger,
		server:        &http.Server{},
		scheduler:     scheduler.NewScheduler(logger, postgres),
		users:         users.NewRegistry(postgres),
		authenticator: &authenticator,
		sessions:      sessions.NewManager("spautofy_session", cfg.SessionKey, time.Hour),
	}
	handler.server.Handler = handler.router()

	return handler
}

func (h *Handler) router() http.Handler {
	router := mux.NewRouter()

	staticAssets := &assetfs.AssetFS{Asset: static.Asset, AssetDir: static.AssetDir, AssetInfo: static.AssetInfo, Prefix: "static"}
	router.PathPrefix("/static").Handler(http.FileServer(staticAssets))

	router.HandleFunc("/", h.renderIndex()).Methods(http.MethodGet)
	router.HandleFunc("/login", h.loginRedirect())
	router.HandleFunc("/login/callback", h.loginCallback())
	router.HandleFunc("/logout", h.logout())

	protected := router.PathPrefix("/account").Subrouter()
	protected.HandleFunc("/{userID:[0-9]+}", h.renderAccount()).Methods(http.MethodGet)
	protected.HandleFunc("/{userID:[0-9]+}", h.updateAccount()).Methods(http.MethodPost)
	protected.Use(h.middlewareAuthenticate, h.middlewareAuthorize)

	router.NotFoundHandler = http.HandlerFunc(h.renderError(http.StatusNotFound))

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

func (h *Handler) StartRunner(ctx context.Context) error {
	h.logger.Log("event", "scheduler.started")
	defer h.logger.Log("event", "scheduler.stopped")

	if err := h.scheduler.Run(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	return nil
}

func (h *Handler) Shutdown(ctx context.Context) error {
	if err := h.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	if err := h.scheduler.Stop(); err != nil {
		return fmt.Errorf("failed to shutdown scheduler: %w", err)
	}

	return nil
}
