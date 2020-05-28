package user

import (
	"github.com/jace-ys/go-library/postgres"
)

type User struct {
}

type Registry struct {
	database *postgres.Client
}

func NewRegistry(postgres *postgres.Client) *Registry {
	return &Registry{
		database: postgres,
	}
}
