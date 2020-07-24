package main

import (
	"context"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/jace-ys/go-library/postgres"
	"golang.org/x/sync/errgroup"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/jace-ys/spautofy/pkg/spautofy"
)

var logger log.Logger

func main() {
	c := parseCommand()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	postgres, err := postgres.NewClient(c.database.ConnectionURL)
	if err != nil {
		exit(err)
	}

	handler := spautofy.NewHandler(logger, &c.spautofy, postgres)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return handler.StartServer(c.port)
	})
	g.Go(func() error {
		return handler.StartMetricsServer(c.metricsPort)
	})
	g.Go(func() error {
		return handler.StartRunner(ctx)
	})
	g.Go(func() error {
		select {
		case <-ctx.Done():
			if err := handler.Shutdown(ctx); err != nil {
				return err
			}
			return ctx.Err()
		}
	})

	if err := g.Wait(); err != nil {
		exit(err)
	}
}

type config struct {
	port        int
	metricsPort int
	spautofy    spautofy.Config
	database    postgres.ClientConfig
}

func parseCommand() *config {
	var c config

	kingpin.Flag("port", "Port for the Spautofy server.").Envar("PORT").Default("8080").IntVar(&c.port)
	kingpin.Flag("metrics-port", "Port for the Spautofy metrics server.").Envar("METRICS_PORT").Default("9090").IntVar(&c.metricsPort)
	kingpin.Flag("base-url", "Base URL for accessing the Spautofy server.").Envar("BASE_URL").Default("http://127.0.0.1:8080").URLVar(&c.spautofy.BaseURL)
	kingpin.Flag("session-store-key", "Authentication key used for the session store.").Envar("SESSION_STORE_KEY").Default("spautofy").StringVar(&c.spautofy.SessionStoreKey)
	kingpin.Flag("spotify-client-id", "Spotify client ID.").Envar("SPOTIFY_CLIENT_ID").Required().StringVar(&c.spautofy.Spotify.ClientID)
	kingpin.Flag("spotify-client-secret", "Spotify client secret.").Envar("SPOTIFY_CLIENT_SECRET").Required().StringVar(&c.spautofy.Spotify.ClientSecret)
	kingpin.Flag("sendgrid-api-key", "API key for accessing the SendGrid API.").Envar("SENDGRID_API_KEY").Required().StringVar(&c.spautofy.SendGrid.APIKey)
	kingpin.Flag("sendgrid-sender-name", "Name to use when sending mail via SendGrid.").Envar("SENDGRID_SENDER_NAME").Required().StringVar(&c.spautofy.SendGrid.SenderName)
	kingpin.Flag("sendgrid-sender-email", "Email to use when sending mail via SendGrid.").Envar("SENDGRID_SENDER_EMAIL").Required().StringVar(&c.spautofy.SendGrid.SenderEmail)
	kingpin.Flag("sendgrid-template-id", "Template ID to use when sending mail via SendGrid.").Envar("SENDGRID_TEMPLATE_ID").Required().StringVar(&c.spautofy.SendGrid.TemplateID)
	kingpin.Flag("database-url", "URL for connecting to Postgres.").Envar("DATABASE_URL").Default("postgres://spautofy:spautofy@127.0.0.1:5432/spautofy").StringVar(&c.database.ConnectionURL)
	kingpin.Parse()

	return &c
}

func exit(err error) {
	logger.Log("event", "app.fatal", "error", err)
	os.Exit(1)
}
