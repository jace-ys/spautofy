package playlist

import (
	"github.com/go-kit/kit/log"
	"github.com/zmb3/spotify"

	"github.com/jace-ys/spautofy/pkg/users"
)

type Creator struct {
	logger        log.Logger
	authenticator *spotify.Authenticator
	users         *users.Registry
}

func NewCreator(logger log.Logger, authenticator *spotify.Authenticator, users *users.Registry) *Creator {
	return &Creator{
		logger:        logger,
		authenticator: authenticator,
		users:         users,
	}
}

func (c *Creator) Run(userID string) func() {
	return func() {
		logger := log.With(c.logger, "user", userID)
		logger.Log("event", "playlist.create.started")

		err := c.CreatePlaylist(userID)
		if err != nil {
			logger.Log("event", "playlist.create.failed")
			return
		}

		logger.Log("event", "playlist.create.finished")
	}
}

func (c *Creator) CreatePlaylist(userID string) error {
	return nil
}
