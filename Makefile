include .env

TARGET = spautofy
SOURCE = cmd/spautofy/main.go
DEPENDENCIES = postgres postgres.init

.PHONY: default run build assets image dependencies test format

default: assets format run

run:
	@echo "==> Executing code.."
	@go run ${SOURCE} \
		--port 8080 \
		--hostname localhost:8080 \
		--session-store-key spautofy \
		--spotify-client-id ${SPOTIFY_CLIENT_ID} \
		--spotify-client-secret ${SPOTIFY_CLIENT_SECRET} \
		--sendgrid-api-key ${SENDGRID_API_KEY} \
		--sendgrid-sender-name Spautofy \
		--sendgrid-sender-email ${SENDGRID_SENDER_EMAIL} \
		--sendgrid-template-id ${SENDGRID_TEMPLATE_ID} \
		--postgres-host 127.0.0.1:5432 \
		--postgres-user spautofy \
		--postgres-password spautofy \
		--postgres-db spautofy

build:
	@echo "==> Compiling code.."
	go build -o ${TARGET} ${SOURCE}

assets:
	@echo "==> Generating assets.."
	go-bindata -modtime 1234567890 -o pkg/web/templates/assets.go -prefix web -pkg templates web/templates/...
	go-bindata -modtime 1234567890 -o pkg/web/static/assets.go -prefix web -pkg static web/static/...

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
	gofmt -w .
