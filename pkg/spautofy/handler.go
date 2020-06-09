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

	"github.com/jace-ys/spautofy/pkg/accounts"
	"github.com/jace-ys/spautofy/pkg/playlists"
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
	users         *users.Registry
	accounts      *accounts.Registry
	scheduler     *scheduler.Scheduler
	playlists     *playlists.Builder
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
		users:         users.NewRegistry(postgres),
		accounts:      accounts.NewRegistry(postgres),
		scheduler:     scheduler.NewScheduler(logger, postgres),
		authenticator: &authenticator,
		sessions:      sessions.NewManager("spautofy_session", cfg.SessionKey, time.Hour),
	}

	handler.server.Handler = handler.router()
	handler.playlists = playlists.NewBuilder(logger, postgres, handler.users, handler.authenticator)

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

	protected := router.PathPrefix("/accounts").Subrouter()
	protected.HandleFunc("/{userID:[0-9]+}", h.renderAccount()).Methods(http.MethodGet)
	protected.HandleFunc("/{userID:[0-9]+}", h.updateAccount()).Methods(http.MethodPost)
	protected.HandleFunc("/{userID:[0-9]+}/unsubscribe", h.deleteAccount()).Methods(http.MethodGet)
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

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	var count int
	var err error
	for {
		count, err = h.loadSchedules(ctx)
		if err == nil {
			break
		}

		h.logger.Log("event", "schedules.load.retried")
		select {
		case <-time.After(15 * time.Second):
			continue
		case <-ctx.Done():
			err = fmt.Errorf("%s: %w", ctx.Err(), err)
			return fmt.Errorf("failed to load schedules: %w", err)
		}
	}

	h.logger.Log("event", "schedules.loaded", "loaded", count)

	if err := h.scheduler.Run(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	return nil
}

func (h *Handler) loadSchedules(ctx context.Context) (int, error) {
	schedules, err := h.scheduler.List(ctx)
	if err != nil {
		return 0, err
	}

	for idx, schedule := range schedules {
		account, err := h.accounts.Get(ctx, schedule.UserID)
		if err != nil {
			return idx, err
		}

		schedule.Spec = account.Schedule
		schedule.Cmd = h.playlists.Run(account.UserID, account.TrackLimit, account.WithEmail)

		_, err = h.scheduler.Create(ctx, schedule)
		if err != nil {
			return idx, err
		}
	}

	return len(schedules), nil
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
