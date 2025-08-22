# Load .env variables
ifneq (,$(wildcard .env))
	include .env
	export $(shell sed 's/=.*//' .env)
endif

# DATABASE
DB_HOST_DEFAULT=localhost
DB_URL=postgres://$(DB_WRITE_USER):$(DB_WRITE_PASSWORD)@$(DB_HOST_DEFAULT):$(DB_WRITE_PORT)/$(DB_WRITE_NAME)?sslmode=disable

# Migration settings
MIGRATE_BIN=~/go/bin/migrate
MIGRATION_DIR=./migration

# Project settings
CMD_DIR=cmd/server

.PHONY: run build tidy fmt test migrate help

## Tidy up Go modules
tidy:
	go mod tidy

## Run the application
run:
	@echo "🚀 Running $(APP_NAME) on port $(APP_PORT) ..."
	go run $(CMD_DIR)/main.go

## Build the application
build:
	@echo "📦 Building $(APP_NAME)..."
	go build -o ${APP_BIN_FILE} $(CMD_DIR)/main.go

run-build:
	@echo "🚀 Running $(APP_NAME) on port $(APP_PORT) ..."
	${APP_BIN_FILE}

# Development Docker commands
docker-up:
	@echo "Building Docker image for development..."
	docker compose \
	-f deploy/docker/docker-compose.yml \
	--env-file .env \
	--project-name grls \
	up --build

# Development Docker down
docker-down:
	@echo "Stopping Docker containers..."
	docker compose \
	-f deploy/docker/docker-compose.yml \
	--env-file .env \
	--project-name grls \
	down -v

# LOG SERVICE
docker-build-prod:
	docker build -t grls-prod -f deploy/docker/Dockerfile .
docker-run-prod:
	docker run --network ps-net --rm -p 50051:50051 grls-prod

# Migrate
# name -> create_{name_table}_table
migrate-create:
	@read -p "Enter migration name: " name; \
	$(MIGRATE_BIN) create -ext sql -dir $(MIGRATION_DIR) -seq $$name

# Run migration up
migrate-up:
	$(MIGRATE_BIN) -path $(MIGRATION_DIR) -database "$(DB_URL)" up

# Run migration down
migrate-down:
	@read -p "Enter migration version: " version; \
	$(MIGRATE_BIN) -path $(MIGRATION_DIR) -database "$(DB_URL)" down $$version

# Run full down (danger)
migrate-down-all:
	$(MIGRATE_BIN) -path $(MIGRATION_DIR) -database "$(DB_URL)" down

# Force version (optional helper)
migrate-force:
	@read -p "Enter target version: " version; \
	$(MIGRATE_BIN) -path $(MIGRATION_DIR) -database "$(DB_URL)" force $$version

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
	pkg/proto/wallet/v1/wallet.proto


## Show help
help:
	@echo "Available commands:"
	@echo "  run       - Run the application"
	@echo "  build     - Build the binary"
	@echo "  tidy      - Clean go.mod"
	@echo "  fmt       - Format code"
	@echo "  test      - Run tests"
	@echo "  migrate   - Run database migrations"
