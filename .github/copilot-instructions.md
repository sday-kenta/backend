# AI Coding Guidelines for Translation Service Backend

## Architecture Overview

This is a **Go Clean Architecture** translation service supporting 4 parallel communication protocols:
- **REST API** (Fiber framework, port 8080)
- **gRPC** (port 8081, protobuf-based)
- **RabbitMQ RPC** (AMQP with request-reply pattern)
- **NATS RPC** (messaging with request-reply pattern)

All protocols share the same business logic via a single `translationUseCase`, enforcing protocol independence. See [internal/app/app.go](internal/app/app.go) for server initialization pattern.

### Layer Structure

```
cmd/app/main.go                    # Entry point, loads config
├─ config/config.go                # Environment-based configuration
└─ internal/app/app.go             # Server initialization & DI
   ├─ internal/controller/         # Protocol handlers (REST, gRPC, RMQ, NATS)
   ├─ internal/usecase/translation # Business logic (translation + storage)
   └─ internal/repo/               # Data access (Postgres + Google Translate API)
```

**Key Pattern**: Dependency injection flows downward through constructors. No global state or service locators. Controllers depend on usecases; usecases depend on repositories.

## Critical Workflows

### Development Setup
```bash
make compose-up           # Start Postgres, RabbitMQ, NATS
make run                  # Run app with migrations
make test                 # Run tests (uses mock generation)
make lint                 # golangci-lint
```

### Code Generation
- **Mocks**: `mockgen` is hooked to `contracts.go` files via `//go:generate`. Run `go generate ./...` after modifying interfaces in [internal/usecase/contracts.go](internal/usecase/contracts.go) or [internal/repo/contracts.go](internal/repo/contracts.go).
- **API Docs**: `swag init` updates Swagger from struct tags; see [internal/controller/restapi/v1/translation.go](internal/controller/restapi/v1/translation.go) for tag examples.
- **Protobuf**: `make proto-v1` generates gRPC bindings from [docs/proto/v1/translation.history.proto](docs/proto/v1/translation.history.proto).

### Testing Pattern
Tests use **table-driven approach** with mocks injected via constructor. Example: [internal/usecase/translation_test.go](internal/usecase/translation_test.go). Always:
1. Create mocks with `NewMock{Interface}(gomock.NewController(t))`
2. Define expected calls: `mockRepo.EXPECT().Method(...).Return(...)`
3. Assert via `testify/require` or `assert`

## Project-Specific Patterns

### Controller Implementation
Each protocol implements the same 2 endpoints (translate + history) with identical business logic flow:
1. Parse/validate input
2. Call usecase method with context
3. Return protocol-specific response (JSON for REST, protobuf for gRPC, etc.)

See [internal/controller/restapi/v1/](internal/controller/restapi/v1/) for REST, [internal/controller/grpc/v1/](internal/controller/grpc/v1/) for gRPC patterns.

### Error Handling
- Use `fmt.Errorf("location - operation - detail: %w", err)` for error wrapping with context.
- Controllers log and translate errors to protocol-specific responses (HTTP status, gRPC codes, etc.).
- Never log sensitive data; use logging context [pkg/logger/logger.go](pkg/logger/logger.go).

### Configuration
All settings load from environment variables via [caarlos0/env](https://github.com/caarlos0/env). Required vars in [config/config.go](config/config.go):
- `APP_NAME`, `APP_VERSION` (required)
- `LOG_LEVEL`, `HTTP_PORT`, `GRPC_PORT` (required)
- `PG_URL`, `PG_POOL_MAX` (database)
- `RMQ_URL`, `RMQ_RPC_SERVER`, `RMQ_RPC_CLIENT` (RabbitMQ)
- `NATS_URL`, `NATS_RPC_SERVER` (NATS)

Missing required vars cause startup failure—intentional for safety.

## Integration Points

### Database
- **Driver**: pgx (PostgreSQL 5.x)
- **Migrations**: golang-migrate in [migrations/](migrations/); run on app startup via [internal/app/migrate.go](internal/app/migrate.go)
- **Pattern**: [internal/repo/persistent/translation_postgres.go](internal/repo/persistent/translation_postgres.go) uses Squirrel query builder
- **Concurrency**: Max pool size configured via `PG_POOL_MAX`

### External API
- **Google Translate**: [internal/repo/webapi/translation_google.go](internal/repo/webapi/translation_google.go) implements `TranslationWebAPI` interface
- No authentication; integrates via go-googletrans wrapper
- All translation requests go through this layer

### Message Queues
- **RabbitMQ**: Request-reply via AMQP exchanges/queues; server router in [internal/controller/amqp_rpc/v1/](internal/controller/amqp_rpc/v1/)
- **NATS**: Request-reply via subjects; server router in [internal/controller/nats_rpc/v1/](internal/controller/nats_rpc/v1/)
- Both use [Request-Reply pattern](https://www.enterpriseintegrationpatterns.com/patterns/messaging/RequestReply.html)

## File Naming & Structure Conventions

- **Controllers**: Mirror protocol structure—`internal/controller/{protocol}/v1/` contains versioned logic
- **Entities**: [internal/entity/](internal/entity/) holds domain models with JSON/proto/db tags
- **Interfaces**: Split by concern—`contracts.go` files contain all interfaces for a layer
- **Tests**: `_test.go` suffix; mocks generated to same package with `_mocks_test.go` suffix

## Validation & Logging

- **Input validation**: Use [go-playground/validator](https://github.com/go-playground/validator) with struct tags; see [internal/controller/restapi/v1/request/translate.go](internal/controller/restapi/v1/request/translate.go)
- **Structured logging**: zerolog with [pkg/logger/logger.go](pkg/logger/logger.go); includes request IDs for tracing across protocols
- **Metrics**: Prometheus metrics optional (toggle via `METRICS_ENABLED`)

## Before Making Changes

1. **Adding an endpoint?** Implement in all 4 controllers: REST, gRPC, RabbitMQ, NATS
2. **Changing entities?** Update [internal/entity/](internal/entity/), then regenerate mocks and swagger
3. **Adding a repo interface?** Update [internal/repo/contracts.go](internal/repo/contracts.go) and regenerate mocks
4. **Modifying config?** Update [config/config.go](config/config.go) and docker-compose env vars
5. **Database changes?** Create migration file in [migrations/](migrations/) with `{timestamp}_description.up/down.sql`

## Testing Command Cheat Sheet

```bash
go generate ./...                           # Regenerate mocks
swag init -g internal/controller/restapi/router.go  # Regenerate Swagger
make proto-v1                               # Regenerate gRPC bindings
go test ./...                               # Run all tests
make compose-up-integration-test            # Run integration tests in Docker
```
