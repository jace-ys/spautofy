package spautofy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	assetfs "github.com/elazarl/go-bindata-assetfs"
	"github.com/etherlabsio/healthcheck"
	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zmb3/spotify"

	"github.com/jace-ys/go-library/postgres"
	"github.com/jace-ys/spautofy/pkg/accounts"
	"github.com/jace-ys/spautofy/pkg/mail"
	"github.com/jace-ys/spautofy/pkg/playlists"
	"github.com/jace-ys/spautofy/pkg/scheduler"
	"github.com/jace-ys/spautofy/pkg/sessions"
	"github.com/jace-ys/spautofy/pkg/users"
	"github.com/jace-ys/spautofy/pkg/web/static"
)

type Config struct {
	BaseURL         *url.URL
	SessionStoreKey string
	Spotify         SpotifyConfig
	SendGrid        mail.SendGridConfig
}

type SpotifyConfig struct {
	ClientID     string
	ClientSecret string
}

type Handler struct {
	logger        log.Logger
	server        *http.Server
	metrics       *http.Server
	database      *postgres.Client
	users         *users.Registry
	accounts      *accounts.Registry
	scheduler     *scheduler.Scheduler
	playlists     *playlists.Registry
	builder       *playlists.BuilderFactory
	authenticator *spotify.Authenticator
	sessions      *sessions.Manager
}

func NewHandler(logger log.Logger, cfg *Config, postgres *postgres.Client) *Handler {
	redirectURL := *cfg.BaseURL
	redirectURL.Path = path.Join(redirectURL.Path, "login/callback")
	authenticator := spotify.NewAuthenticator(redirectURL.String(), scopes...)
	authenticator.SetAuthInfo(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret)

	handler := &Handler{
		logger:        logger,
		server:        &http.Server{},
		metrics:       &http.Server{},
		database:      postgres,
		users:         users.NewRegistry(postgres),
		accounts:      accounts.NewRegistry(postgres),
		scheduler:     scheduler.NewScheduler(logger, postgres),
		playlists:     playlists.NewRegistry(postgres),
		authenticator: &authenticator,
		sessions:      sessions.NewManager("spautofy_session", cfg.SessionStoreKey, time.Hour),
	}

	handler.server.Handler = handler.router()

	mailer := mail.NewSendGridMailer(&cfg.SendGrid)
	handler.builder = playlists.NewBuilderFactory(cfg.BaseURL, mailer, handler.playlists, handler.users, handler.authenticator)

	return handler
}

func (h *Handler) router() http.Handler {
	router := mux.NewRouter()

	staticAssets := &assetfs.AssetFS{Asset: static.Asset, AssetDir: static.AssetDir, AssetInfo: static.AssetInfo}
	router.PathPrefix("/static").Handler(http.FileServer(staticAssets))

	router.HandleFunc("/", h.renderIndex()).Methods(http.MethodGet)
	router.HandleFunc("/login", h.loginRedirect())
	router.HandleFunc("/login/callback", h.loginCallback())
	router.HandleFunc("/logout", h.logout())

	accounts := router.PathPrefix("/accounts/{userID:[0-9]+}").Subrouter()
	accounts.Use(h.middlewareAuthenticate, h.middlewareAuthorize)
	accounts.HandleFunc("", h.renderAccount()).Methods(http.MethodGet)
	accounts.HandleFunc("", h.updateAccount()).Methods(http.MethodPost)
	accounts.HandleFunc("/unsubscribe", h.deleteAccount()).Methods(http.MethodGet)

	playlists := accounts.PathPrefix("/playlists/{playlistName}").Subrouter()
	playlists.HandleFunc("", h.renderPlaylist()).Methods(http.MethodGet)
	playlists.HandleFunc("", h.createPlaylist()).Methods(http.MethodPost)

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

func (h *Handler) StartMetricsServer(port int) error {
	h.logger.Log("event", "metrics.started", "port", port)
	defer h.logger.Log("event", "metrics.stopped")

	router := mux.NewRouter()

	router.Handle("/metrics", promhttp.Handler())

	router.Handle("/health", healthcheck.Handler(
		healthcheck.WithChecker(
			"database", healthcheck.CheckerFunc(
				func(ctx context.Context) error {
					return h.database.DB.DB.PingContext(ctx)
				},
			),
		),
	))

	router.HandleFunc("/crons", func(w http.ResponseWriter, r *http.Request) {
		entries := h.scheduler.ListCronEntries()

		data, err := json.Marshal(entries)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}

		w.Write(data)
	})

	h.metrics.Handler = router
	h.metrics.Addr = fmt.Sprintf(":%d", port)

	if err := h.metrics.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
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

		builder, err := h.builder.NewBuilder(ctx, h.logger, account.UserID)
		if err != nil {
			return idx, err
		}

		schedule.Spec = account.Schedule
		schedule.Cmd = builder.Run(account.TrackLimit, account.WithConfirm)

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

	if err := h.metrics.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown metrics server: %w", err)
	}

	if err := h.scheduler.Stop(); err != nil {
		return fmt.Errorf("failed to shutdown scheduler: %w", err)
	}

	return nil
}
