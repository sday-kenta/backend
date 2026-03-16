ifneq ($(wildcard .env),)
include .env
export
else
$(warning WARNING: .env file not found! Using .env.example)
include .env.example
export
endif

BASE_STACK = docker compose -f docker-compose.yml

.PHONY: help compose-up compose-down swag-v1 deps deps-audit format run linter-golangci test mock migrate-create migrate-up bin-deps pre-commit

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

compose-up: ## Run docker compose stack
	$(BASE_STACK) up --build -d

compose-down: ## Stop docker compose stack
	$(BASE_STACK) down --remove-orphans

swag-v1: ## Regenerate swagger
	swag init --generalInfo main.go --dir ./cmd/app,./internal/controller/restapi,./internal/entity,./internal/usecase,./internal/repo --parseInternal

deps: ## deps tidy + verify
	go mod tidy && go mod verify

deps-audit: ## check dependencies vulnerabilities
	govulncheck ./...

format: ## Run code formatter
	gofumpt -l -w .
	gci write . --skip-generated -s standard -s default

run: ## Run application with migrations init
	go mod download && \
	CGO_ENABLED=0 go run -tags migrate ./cmd/app

linter-golangci: ## check by golangci linter
	golangci-lint run

test: ## run tests
	go test -v -race -covermode atomic -coverprofile=coverage.txt ./internal/... ./pkg/...

mock: ## run mockgen
	mockgen -source ./internal/repo/contracts.go -package usecase_test > ./internal/usecase/mocks_repo_test.go
	mockgen -source ./internal/usecase/contracts.go -package usecase_test > ./internal/usecase/mocks_usecase_test.go

migrate-create: ## create new migration
	migrate create -ext sql -dir migrations '$(word 2,$(MAKECMDGOALS))'

migrate-up: ## migration up
	migrate -path migrations -database '$(PG_URL)?sslmode=disable' up

bin-deps: ## install tools
	go install tool
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate

pre-commit: swag-v1 mock format linter-golangci test ## run pre-commit