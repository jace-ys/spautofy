package main

import (
	"context"
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/go-kit/kit/log"
	"github.com/jace-ys/go-library/postgres"
	"golang.org/x/sync/errgroup"

	"github.com/jace-ys/spautofy/pkg/spautofy"
)

var logger log.Logger

func main() {
	c := parseCommand()

	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	postgresClient, err := postgres.NewClient(c.database.Host, c.database.User, c.database.Password, c.database.Database)
	if err != nil {
		exit(err)
	}

	handler := spautofy.NewHandler(logger, &c.spautofy, postgresClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return handler.StartServer(c.port)
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
	port     int
	spautofy spautofy.Config
	database postgres.ClientConfig
}

func parseCommand() *config {
	var c config

	kingpin.Flag("port", "Port for the Spautofy server.").Envar("PORT").Default("8080").IntVar(&c.port)
	kingpin.Flag("redirect-host", "Host for the Spotify OAuth redirect URI.").Envar("REDIRECT_HOST").Default("localhost:8080").StringVar(&c.spautofy.RedirectHost)
	kingpin.Flag("session-key", "Authentication key used for the session store.").Envar("SESSION_KEY").Default("spautofy").StringVar(&c.spautofy.SessionKey)
	kingpin.Flag("spotify-client", "Spotify client ID.").Envar("SPOTIFY_CLIENT_ID").Required().StringVar(&c.spautofy.ClientID)
	kingpin.Flag("spotify-secret", "Spotify client secret.").Envar("SPOTIFY_CLIENT_SECRET").Required().StringVar(&c.spautofy.Secret)
	kingpin.Flag("postgres-host", "Host for connecting to Postgres.").Envar("POSTGRES_HOST").Default("127.0.0.1:5432").StringVar(&c.database.Host)
	kingpin.Flag("postgres-user", "User for connecting to Postgres.").Envar("POSTGRES_USER").Default("postgres").StringVar(&c.database.User)
	kingpin.Flag("postgres-password", "Password for connecting to Postgres.").Envar("POSTGRES_PASSWORD").Required().StringVar(&c.database.Password)
	kingpin.Flag("postgres-db", "Database for connecting to Postgres.").Envar("POSTGRES_DB").Default("postgres").StringVar(&c.database.Database)
	kingpin.Parse()

	return &c
}

func exit(err error) {
	logger.Log("event", "app.fatal", "error", err)
	os.Exit(1)
}
