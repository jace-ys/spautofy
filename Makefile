-include .env

TARGET = spautofy
SOURCE = cmd/spautofy/main.go
DEPENDENCIES = postgres postgres.init

.PHONY: default run build image dependencies test format

default: format run

run:
	@echo "==> Executing code.."
	@go run ${SOURCE} \
		--port 8080 \
		--metrics-port 9090 \
		--base-url http://localhost:8080 \
		--session-store-key spautofy \
		--spotify-client-id ${SPOTIFY_CLIENT_ID} \
		--spotify-client-secret ${SPOTIFY_CLIENT_SECRET} \
		--sendgrid-api-key ${SENDGRID_API_KEY} \
		--sendgrid-sender-name ${SENDGRID_SENDER_NAME} \
		--sendgrid-sender-email ${SENDGRID_SENDER_EMAIL} \
		--sendgrid-template-id ${SENDGRID_TEMPLATE_ID} \
		--database-url postgres://spautofy:spautofy@127.0.0.1:5432/spautofy?sslmode=disable

build:
	@echo "==> Compiling code.."
	go build -o ${TARGET} ${SOURCE}

image:
	@echo "==> Building image.."
	docker-compose build spautofy

dependencies:
	@echo "==> Starting auxiliary containers.."
	docker-compose up -d ${DEPENDENCIES}

test:
	@echo "==> Running tests.."
	go test -v ./...

format:
	@echo "==> Formatting code.."
	go fmt ./...
