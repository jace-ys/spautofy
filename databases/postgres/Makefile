.PHONY: migration

migration:
	@echo "==> Generating new migration files.."
	migrate create -ext sql -dir migrations ${NAME}

psql:
	@echo "==> Starting psql interactive terminal.."
	psql -U ${POSTGRES_USER}
