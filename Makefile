include .env

TARGET = spautofy
SOURCE = cmd/spautofy/main.go
DEPENDENCIES = postgres postgres.init

.PHONY: all build run proxy dependencies test format

all: format run

build:
	@echo "==> Compiling code.."
	go build -o ${TARGET} ${SOURCE}

run:
	@echo "==> Executing code.."
	@go run ${SOURCE} \
		--port 8080 \
		--spotify-client ${SPOTIFY_CLIENT_ID} \
		--spotify-secret ${SPOTIFY_CLIENT_SECRET} \
		--postgres-host 127.0.0.1:5432 \
		--postgres-user spautofy \
		--postgres-password spautofy \
		--postgres-db spautofy

image:
	@docker-compose build spautofy

dependencies:
	@echo "==> Starting auxiliary containers.."
	docker-compose up -d ${DEPENDENCIES}

test:
	@echo "==> Running tests.."
	go test -v ./...

format:
	@echo "==> Formatting code.."
	gofmt -w .
