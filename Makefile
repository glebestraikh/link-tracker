COVERAGE_FILE ?= coverage.out

# Get all directories in cmd/ as available modules
MODULES := $(notdir $(wildcard cmd/*))

# Migrator binaries: cmd/<svc>/migrate/main.go → bin/<svc>-migrate
MIGRATORS := bot scrapper

# Help target - display usage information
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  \033[36mmake build\033[0m - Build all modules ($(MODULES)) and migrators ($(MIGRATORS))"
	@$(foreach mod,$(MODULES),echo "  \033[36mmake build_$(mod)\033[0m - Build $(mod) module";)
	@$(foreach m,$(MIGRATORS),echo "  \033[36mmake build_$(m)_migrate\033[0m - Build $(m) migrator";)
	@echo "  \033[36mmake test\033[0m - Run all tests"

.PHONY: build
build: $(addprefix build_,$(MODULES)) $(addprefix build_,$(addsuffix _migrate,$(MIGRATORS)))

.PHONY: $(addprefix build_,$(MODULES))
$(addprefix build_,$(MODULES)):
	@modulename=$(subst build_,,$@); \
	echo "Building module: $$modulename"; \
	mkdir -p bin; \
	go build -o ./bin/$$modulename ./cmd/$$modulename

.PHONY: $(addprefix build_,$(addsuffix _migrate,$(MIGRATORS)))
$(addprefix build_,$(addsuffix _migrate,$(MIGRATORS))):
	@svc=$(subst build_,,$(subst _migrate,,$@)); \
	echo "Building migrator: $$svc-migrate"; \
	mkdir -p bin; \
	go build -o ./bin/$$svc-migrate ./cmd/$$svc/migrate

## test: run all tests
.PHONY: test
test:
	@go test -coverpkg='gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/...' --race -count=1 -coverprofile='$(COVERAGE_FILE)' ./...
	@go tool cover -func='$(COVERAGE_FILE)' | grep ^total | tr -s '\t'

## integration-test: run integration tests with logs (requires Docker for Testcontainers)
.PHONY: integration-test
integration-test:
	@go test -tags=integration ./test/integration/... -count=1 -v -timeout 5m

## integration-test-quiet: run integration tests quietly
.PHONY: integration-test-quiet
integration-test-quiet:
	@go test -tags=integration ./test/integration/... -count=1 -timeout 5m

## migrate-up: apply migrations to bot and scrapper databases (uses .env files)
.PHONY: migrate-up
migrate-up: build_bot_migrate build_scrapper_migrate
	@. ./.env.bot && DATABASE_URL=$$APP_DATABASE_URL ./bin/bot-migrate -direction up
	@. ./.env.scrapper && DATABASE_URL=$$APP_DATABASE_URL ./bin/scrapper-migrate -direction up

## migrate-down: revert migrations
.PHONY: migrate-down
migrate-down: build_bot_migrate build_scrapper_migrate
	@. ./.env.bot && DATABASE_URL=$$APP_DATABASE_URL ./bin/bot-migrate -direction down
	@. ./.env.scrapper && DATABASE_URL=$$APP_DATABASE_URL ./bin/scrapper-migrate -direction down

## compose-up: bring up postgres + services via docker-compose (runs migrators automatically)
.PHONY: compose-up
compose-up:
	@docker compose up -d --build

## compose-down: stop docker-compose stack
.PHONY: compose-down
compose-down:
	@docker compose down
