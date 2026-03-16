# Merge notes

This repository was assembled by combining the maps and users services into one Fiber application.

## What was merged directly
- maps domain files (`internal/entity/category.go`, `internal/entity/geo.go`, `internal/usecase/category`, `internal/usecase/geo`, `internal/repo/persistent/category_postgres.go`, `internal/repo/persistent/geo_postgres.go`, `internal/repo/webapi`)
- users domain files (`internal/entity/user.go`, `internal/usecase/user`, `internal/repo/persistent/user_postgres.go`, `internal/usererr`, `internal/controller/restapi/v1/user.go`)
- migrations from both services into one `migrations/` directory

## Files that had to be rebuilt as a single shared version
- `config/config.go`
- `internal/app/app.go`
- `internal/repo/contracts.go`
- `internal/usecase/contracts.go`
- `internal/controller/restapi/router.go`
- `cmd/app/main.go`
- `Dockerfile`
- `docker-compose.yml`
- `Makefile`

## Important unresolved architectural decisions
1. **Authorization model is still split**
   - maps admin endpoints still rely on `X-User-Role: admin`
   - users domain has roles stored in DB but no shared auth middleware

2. **Avatar upload still owns infra in the HTTP handler**
   - `internal/controller/restapi/v1/user.go` still reads S3/MinIO env vars directly and uploads from the handler
   - cleaner long-term option: move storage behind a repo/service abstraction and inject it from `app.Run`

3. **Swagger should be regenerated**
   - docs were merged statically so the repo has a single docs package now
   - the next clean step is `make swag-v1` in a working Go 1.25 environment

4. **Compose was normalized around PostGIS**
   - maps requires PostGIS, so the merged stack uses PostGIS for all domains
   - users works on PostGIS-backed PostgreSQL too, but this is still a conscious infra choice

## Decisions applied in the current merge pass
- authorization changes were intentionally postponed; maps admin handlers still use `X-User-Role` temporarily
- PostGIS remains the single shared database engine for both domains
- CORS stays global because Swagger UI / browser clients may call the API from another origin
- Prometheus metrics stay optional behind `METRICS_ENABLED`; when enabled they expose `/metrics` for the whole merged app
- swagger annotations for users + maps live together under `internal/controller/restapi/v1/*`; regenerate docs with `make swag-v1` after local verification
- `tool (...)` was cleaned from old proto/grpc generators that are no longer used by the merged service
