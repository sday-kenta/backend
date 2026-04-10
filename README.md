# backend

Combined Go/Fiber backend for the **"Сознательный гражданин"** project.

The service already includes users, categories, maps/geocoding, and the incidents/messages domain. It also exposes Swagger, Prometheus metrics, MinIO/S3-based file storage, and PostgreSQL/PostGIS integration.

## Current status

Implemented now:
- users and profile editing;
- login by `login` / `email` / `phone`;
- email verification code sending and verification;
- password reset code sending;
- categories CRUD;
- maps search and reverse geocoding via OSM/Nominatim + PostGIS zones;
- avatar upload to MinIO/S3;
- incidents CRUD;
- incident photo upload;
- draft / published incident flow;
- incident document rendering for download / print / email.

Important limitations of the current MVP:
- there is **no JWT-based auth yet**;
- temporary access control is based on headers;
- `X-User-Role: admin` is still used for admin-only routes;
- incident ownership is currently identified by `X-User-ID`;
- password reset flow is still partial: only sending the code is implemented.

## Stack

- Go 1.25
- Fiber
- PostgreSQL + PostGIS
- Swagger / Swaggo
- MinIO via S3 API
- OpenStreetMap / Nominatim
- Docker Compose

## Project structure

- `cmd/app` - application entrypoint
- `config` - environment-driven config
- `internal/app` - DI/wiring
- `internal/controller/restapi` - HTTP routing and handlers
- `internal/entity` - domain entities
- `internal/repo` - persistent and external repositories
- `internal/usecase` - business logic
- `migrations` - SQL migrations
- `docs` - generated Swagger files
- `pkg/objectstorage` - shared MinIO/S3 helper
- `pkg/mailsender` - email sending helpers

## Quick start

### 1. Configure environment

Create `.env` from `.env.example`:

```bash
cp .env.example .env
```

At minimum, review these values:
- `PG_URL`
- `AWS_S3_BUCKET`
- `AWS_S3_ENDPOINT`
- `AVATAR_BASE_URL`
- `INCIDENT_MEDIA_BASE_URL`
- `SMTP_MAIL`
- `SMTP_CODE`
- `SMTP_HOST`
- `SMTP_PORT`

`SMTP_MAIL` and `SMTP_CODE` are required for:
- `POST /v1/users/email-code/send`
- `POST /v1/users/password-reset/send-code`
- `POST /v1/incidents/{id}/document/email`

SMTP server can be customized with:
- `SMTP_HOST` (default: `smtp.mail.ru`)
- `SMTP_PORT` (default: `587`)

### 2. Run the full stack

```bash
make compose-up
```

This starts:
- `db` - PostgreSQL/PostGIS
- `migrator` - SQL migrations
- `db-seed` - initial Samara geo seed
- `minio` - S3-compatible object storage
- `mc-init` - bucket creation + public download policy
- `app` - HTTP API

### 3. Useful URLs

- API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Healthcheck: `http://localhost:8080/healthz`
- Metrics: `http://localhost:8080/metrics`
- MinIO API: `http://localhost:9000`
- MinIO Console: `http://localhost:9001`

### 4. Stop the stack

```bash
make compose-down
```

## Environment variables

Main variables from `.env.example`:

```env
APP_NAME=backend
APP_VERSION=1.0.0
HTTP_PORT=8080
HTTP_USE_PREFORK_MODE=false
LOG_LEVEL=debug

POSTGRES_USER=user
POSTGRES_PASSWORD=postgres
POSTGRES_DB=backend
PG_POOL_MAX=10
PG_URL=postgres://user:postgres@db:5432/backend

METRICS_ENABLED=true
SWAGGER_ENABLED=true

GEO_CACHE_RADIUS_METERS=20
GEO_MAX_CITY_ATTEMPTS=4

NOMINATIM_BASE_URL=https://nominatim.openstreetmap.org
NOMINATIM_USER_AGENT=sday-kenta/1.0
NOMINATIM_EMAIL=
NOMINATIM_ACCEPT_LANGUAGE=ru
NOMINATIM_COUNTRY_CODES=ru
NOMINATIM_SEARCH_LIMIT=5
NOMINATIM_REVERSE_ZOOM=18
NOMINATIM_TIMEOUT=5s

AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=minioadmin
AWS_SECRET_ACCESS_KEY=minioadmin123
AWS_S3_BUCKET=avatars
AWS_S3_ENDPOINT=http://minio:9000
AVATAR_BASE_URL=http://localhost:9000/avatars
INCIDENT_MEDIA_BASE_URL=http://localhost:9000/avatars

SMTP_MAIL=
SMTP_CODE=
SMTP_HOST=smtp.mail.ru
SMTP_PORT=587
```

Notes:
- by default both avatars and incident photos are stored in the same S3 bucket;
- incident photos use keys like `incidents/<incident_id>/<timestamp>.<ext>`;
- file URLs are built from `AVATAR_BASE_URL` and `INCIDENT_MEDIA_BASE_URL`.

## HTTP API

Base path: `/v1`

### Service routes

- `GET /healthz`
- `GET /metrics`
- `GET /swagger/index.html`

### Auth

- `POST /v1/auth/login`
- `POST /v1/auth/register`

### Users

- `POST /v1/users` - admin only
- `GET /v1/users`
- `GET /v1/users/{id}`
- `PUT /v1/users/{id}`
- `DELETE /v1/users/{id}`
- `POST /v1/users/{id}/avatar`
- `POST /v1/users/email-code/send`
- `POST /v1/users/email-code/verify`
- `POST /v1/users/password-reset/send-code`

### Categories

- `GET /v1/categories`
- `GET /v1/categories/{id}`
- `POST /v1/categories` - admin only
- `PATCH /v1/categories/{id}` - admin only
- `DELETE /v1/categories/{id}` - admin only

### Maps

- `GET /v1/maps/reverse`
- `GET /v1/maps/search`
- `POST /v1/maps/reload-cities` - admin only

### Incidents

- `POST /v1/incidents`
- `GET /v1/incidents`
- `GET /v1/incidents/{id}`
- `PATCH /v1/incidents/{id}`
- `DELETE /v1/incidents/{id}`
- `GET /v1/my/incidents`
- `POST /v1/incidents/{id}/photos`
- `DELETE /v1/incidents/{id}/photos/{photoId}`
- `GET /v1/incidents/{id}/document/download`
- `GET /v1/incidents/{id}/document/print`
- `POST /v1/incidents/{id}/document/email`

## Access model

Current temporary headers:

- `X-User-Role: admin` - for admin-only routes
- `X-User-ID: <id>` - for incident ownership and "my incidents" routes

### Routes that require `X-User-ID`

- `POST /v1/incidents`
- `GET /v1/my/incidents`
- `PATCH /v1/incidents/{id}`
- `DELETE /v1/incidents/{id}`
- `POST /v1/incidents/{id}/photos`
- `DELETE /v1/incidents/{id}/photos/{photoId}`
- `GET /v1/incidents/{id}/document/download`
- `GET /v1/incidents/{id}/document/print`
- `POST /v1/incidents/{id}/document/email`

### Routes that require `X-User-Role: admin`

- `POST /v1/categories`
- `PATCH /v1/categories/{id}`
- `DELETE /v1/categories/{id}`
- `POST /v1/maps/reload-cities`

## Incident flow

### Address model

For MVP, an incident keeps both:
- coordinates: `latitude`, `longitude`;
- address fields: `city`, `street`, `house`, `address_text`.

This keeps map behavior, filtering, and document generation simple without introducing a separate address table.

### Incident statuses

- `draft`
- `review`
- `published`

Behavior:
- `GET /v1/incidents` returns **published only** for regular users and anonymous requests;
- for admins, `GET /v1/incidents` returns `published + review` by default;
- admins may repeat the `status` query parameter, for example `?status=review&status=published`, to control which statuses are returned;
- `GET /v1/my/incidents` returns current user's incidents, including drafts and review items;
- draft and review details are visible only to the author or admin;
- when a regular user sends `status=published` in `POST /v1/incidents` or `PATCH /v1/incidents/{id}`, the backend stores `review` instead;
- an admin may store `published` directly.
- an admin cannot edit another user's `draft` incident through `PATCH /v1/incidents/{id}`.
- only the incident author may upload photos; an admin may still delete another user's incident or photos.

### Incident photos

Photo upload uses `multipart/form-data` with repeated `photos` fields.

Restrictions in handler:
- max 5 MB per file;
- allowed extensions: `.png`, `.jpg`, `.jpeg`.

### Incident document actions

The project treats the final document as a rendered representation built from incident data.

Available actions:
- `GET /v1/incidents/{id}/document/download` - returns HTML with `Content-Disposition: attachment`
- `GET /v1/incidents/{id}/document/print` - returns HTML with `Content-Disposition: inline`
- `POST /v1/incidents/{id}/document/email` - sends the rendered HTML document by email

## Development commands

```bash
make help
make compose-up
make compose-down
make run
make test
make format
make swag-v1
make migrate-up
```

## Swagger

Swagger is available in two forms:
- source annotations in handlers;
- generated files in `docs/`.

If annotations were changed, regenerate docs with:

```bash
make swag-v1
```

## Notes

- README is intentionally aligned with the current code and migrations.
- If README and generated Swagger disagree, treat the **code + migrations + handler annotations** as the source of truth.
- `README_RU.md` contains the same overview in Russian.
