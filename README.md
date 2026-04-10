# backend

Combined Go/Fiber backend for the **"Сознательный гражданин"** project.

The service already includes users, categories, maps/geocoding, incidents, feedback email, Swagger, Prometheus metrics, MinIO/S3-based file storage, PostgreSQL/PostGIS integration, and Firebase Cloud Messaging push delivery.

## Current status

Implemented now:
- Bearer JWT auth with public `POST /v1/auth/register` and `POST /v1/auth/login`;
- optional admin bootstrap via environment variables;
- user management, profile editing, and avatar upload;
- email verification code sending and verification;
- password reset by email code;
- admin-only user creation without email confirmation, with `role` and `is_blocked`;
- categories CRUD plus icon upload/delete;
- maps search and reverse geocoding via OSM/Nominatim + PostGIS zones;
- incidents CRUD with moderation flow `draft / review / published`;
- incident photo upload with ownership checks;
- incident document rendering for download / print / email;
- public feedback email endpoint;
- FCM push device registration and incident status notifications;
- Swagger and Prometheus integration.

Important limitations of the current MVP:
- push delivery is synchronous best-effort; there is no queue, RMQ, or Redis layer;
- authentication currently uses a single access JWT, without refresh tokens;
- PWA push still requires Firebase Web SDK, VAPID key, and a service worker on the frontend;
- Firebase service account credentials must be provided locally or via deployment secrets and must never be committed to git.

## Stack

- Go 1.25
- Fiber
- PostgreSQL + PostGIS
- Swagger / Swaggo
- MinIO via S3 API
- OpenStreetMap / Nominatim
- Firebase Cloud Messaging
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
- `pkg/authjwt` - JWT generation and parsing
- `pkg/mailsender` - email sending helpers
- `pkg/objectstorage` - shared MinIO/S3 helper
- `pkg/pushclient` - FCM HTTP v1 sender

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
- `CATEGORY_MEDIA_BASE_URL`
- `JWT_SECRET`
- `SMTP_MAIL`
- `SMTP_CODE`
- `SMTP_HOST`
- `SMTP_PORT`
- `FCM_ENABLED`
- `FCM_CREDENTIALS_FILE`

If you want the app to create the first admin automatically on startup, also configure:
- `ADMIN_BOOTSTRAP_ENABLED=true`
- `ADMIN_BOOTSTRAP_LOGIN`
- `ADMIN_BOOTSTRAP_EMAIL`
- `ADMIN_BOOTSTRAP_PASSWORD`
- `ADMIN_BOOTSTRAP_PHONE`

For local push testing, place the Firebase service account JSON at the path from `FCM_CREDENTIALS_FILE` (by default `./firebase-service-account.json`) and keep it untracked. In production, mount it as a secret file and point `FCM_CREDENTIALS_FILE` to that path.

`SMTP_MAIL` and `SMTP_CODE` are required for:
- `POST /v1/users/email-code/send`
- `POST /v1/users/password-reset/send-code`
- `POST /v1/users/password-reset/reset`
- `POST /v1/incidents/{id}/document/email`
- `POST /v1/feedback`

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
LOG_PRETTY=true
HTTP_LOG_HEADERS=true
HTTP_LOG_BODY=true
HTTP_LOG_BODY_MAX_BYTES=4096

POSTGRES_USER=user
POSTGRES_PASSWORD=postgres
POSTGRES_DB=backend
PG_POOL_MAX=10
PG_URL=postgres://user:postgres@db:5432/backend

METRICS_ENABLED=true
SWAGGER_ENABLED=true
SWAGGER_HOST=localhost:8080
SWAGGER_BASE_PATH=/v1
SWAGGER_SCHEMES=http

AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=minioadmin
AWS_SECRET_ACCESS_KEY=minioadmin123
AWS_S3_BUCKET=avatars
AWS_S3_ENDPOINT=http://minio:9000
AVATAR_BASE_URL=http://localhost:9000/avatars
INCIDENT_MEDIA_BASE_URL=http://localhost:9000/avatars
CATEGORY_MEDIA_BASE_URL=http://localhost:9000/avatars

ADMIN_BOOTSTRAP_ENABLED=false
ADMIN_BOOTSTRAP_LOGIN=admin
ADMIN_BOOTSTRAP_EMAIL=admin@mail.ru
ADMIN_BOOTSTRAP_PASSWORD=admin
ADMIN_BOOTSTRAP_PHONE=+79990000000

JWT_SECRET=dev-secret-change-me
JWT_TTL=24h
JWT_ISSUER=backend

FCM_ENABLED=false
FCM_CREDENTIALS_FILE=./firebase-service-account.json
FCM_TIMEOUT=5s

SMTP_MAIL=address@example.com
SMTP_CODE=XXXX XXXX XXXX XXXX
SMTP_HOST=smtp.mail.ru
SMTP_PORT=587
```

Notes:
- by default avatars, incident photos, and category icons can share the same S3 bucket;
- incident photos use keys like `incidents/<incident_id>/<timestamp>.<ext>`;
- file URLs are built from `AVATAR_BASE_URL`, `INCIDENT_MEDIA_BASE_URL`, and `CATEGORY_MEDIA_BASE_URL`;
- when `FCM_ENABLED=false`, push registration endpoints still work at the persistence layer, but actual delivery is disabled.

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

- `POST /v1/users` - admin only, creates user immediately without email confirmation
- `GET /v1/users` - admin only
- `GET /v1/users/{id}` - self or admin
- `PUT /v1/users/{id}` - self or admin
- `DELETE /v1/users/{id}` - admin only
- `POST /v1/users/{id}/avatar` - self or admin
- `POST /v1/users/email-code/send`
- `POST /v1/users/email-code/verify`
- `POST /v1/users/password-reset/send-code`
- `POST /v1/users/password-reset/reset`

### Categories

- `GET /v1/categories`
- `GET /v1/categories/{id}`
- `POST /v1/categories` - admin only
- `PATCH /v1/categories/{id}` - admin only
- `POST /v1/categories/{id}/icon` - admin only
- `DELETE /v1/categories/{id}/icon` - admin only
- `DELETE /v1/categories/{id}` - admin only

### Maps

- `GET /v1/maps/reverse`
- `GET /v1/maps/search`
- `POST /v1/maps/reload-cities` - admin only

### Incidents

- `POST /v1/incidents` - auth required
- `GET /v1/incidents`
- `GET /v1/incidents/{id}`
- `PATCH /v1/incidents/{id}` - auth required
- `DELETE /v1/incidents/{id}` - auth required
- `GET /v1/my/incidents` - auth required
- `POST /v1/incidents/{id}/photos` - auth required
- `DELETE /v1/incidents/{id}/photos/{photoId}` - auth required
- `GET /v1/incidents/{id}/document/download` - auth required
- `GET /v1/incidents/{id}/document/print` - auth required
- `POST /v1/incidents/{id}/document/email` - auth required

### Push

- `POST /v1/push/devices` - auth required
- `DELETE /v1/push/devices/{deviceId}` - auth required

### Feedback

- `POST /v1/feedback`

## Access model

The `/v1` API group uses optional Bearer JWT parsing. Public routes can be called without `Authorization`, while protected routes require:

```http
Authorization: Bearer <access-token>
```

Main rules:
- `POST /v1/auth/register` is the public self-registration flow with email confirmation;
- `POST /v1/users` is a separate admin-only flow for immediate user creation;
- `GET /v1/incidents` is public, but the response depends on role if a valid JWT is provided;
- `GET /v1/incidents/{id}` is public only for `published` incidents; `draft` and `review` require author or admin access;
- `GET /v1/users/{id}`, `PUT /v1/users/{id}`, and `POST /v1/users/{id}/avatar` are limited to self or admin;
- admin-only routes use role from JWT, not custom headers.

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
- if status is omitted in `POST /v1/incidents`, the backend stores `review`;
- `GET /v1/incidents` returns only `published` for regular users and anonymous requests;
- for admins, `GET /v1/incidents` returns `published + review` by default;
- admins may repeat the `status` query parameter, for example `?status=review&status=published`, to choose which statuses are returned;
- `GET /v1/incidents` and `GET /v1/my/incidents` also support `category_id`;
- `GET /v1/my/incidents` returns the current user's incidents and supports `status=draft|review|published|all`;
- `draft` and `review` details are visible only to the author or admin;
- when a regular user sends `status=published` in `POST /v1/incidents` or `PATCH /v1/incidents/{id}`, the backend stores `review` instead;
- an admin may store `published` directly;
- an admin cannot edit another user's `draft` incident through `PATCH /v1/incidents/{id}`;
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

## Push notifications

Push delivery is implemented through FCM HTTP v1.

Backend behavior:
- `POST /v1/push/devices` registers or refreshes a device token for the authenticated user;
- `DELETE /v1/push/devices/{deviceId}` detaches the client-generated device ID;
- supported platforms are `android`, `ios`, and `web`; `pwa` is accepted and normalized to `web`;
- notification delivery is triggered on incident status changes:
  - non-published -> `published`
  - `review` -> `draft` when the actor is not the incident author

For local development:
- place the Firebase service account JSON at `FCM_CREDENTIALS_FILE`;
- keep it out of git;
- enable delivery with `FCM_ENABLED=true`.

For a PWA frontend:
- use Firebase Web SDK;
- generate a web push token with a VAPID key and service worker;
- send that token to `POST /v1/push/devices` with `platform=web`.

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
