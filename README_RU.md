# backend

Объединённый Go/Fiber backend для проекта **«Сознательный гражданин»**.

Сервис уже включает домены users, categories, maps/geocoding и incidents/messages. Также доступны Swagger, Prometheus metrics, загрузка файлов в MinIO/S3 и PostgreSQL/PostGIS.

## Текущее состояние

Сейчас реализовано:
- пользователи и редактирование профиля;
- логин по `login` / `email` / `phone`;
- отправка и проверка email-кода;
- отправка кода для сброса пароля;
- CRUD для рубрикатора;
- поиск адресов и reverse geocoding через OSM/Nominatim + PostGIS-зоны;
- загрузка аватара в MinIO/S3;
- CRUD для инцидентов;
- загрузка фотографий инцидента;
- flow `draft / published` для инцидентов;
- генерация HTML-документа обращения для скачивания / печати / отправки на email.

Важные ограничения текущего MVP:
- **JWT-авторизации пока нет**;
- доступ временно контролируется заголовками;
- для admin-only ручек всё ещё используется `X-User-Role: admin`;
- автор инцидента пока определяется через `X-User-ID`;
- reset password пока неполный: реализована только отправка кода.

## Стек

- Go 1.25
- Fiber
- PostgreSQL + PostGIS
- Swagger / Swaggo
- MinIO через S3 API
- OpenStreetMap / Nominatim
- Docker Compose

## Структура проекта

- `cmd/app` - точка входа приложения
- `config` - конфигурация из переменных окружения
- `internal/app` - wiring / DI
- `internal/controller/restapi` - HTTP-маршруты и хендлеры
- `internal/entity` - доменные сущности
- `internal/repo` - persistent и external repositories
- `internal/usecase` - бизнес-логика
- `migrations` - SQL-миграции
- `docs` - сгенерированные swagger-файлы
- `pkg/objectstorage` - общий helper для MinIO/S3
- `pkg/mailsender` - helpers для отправки email

## Быстрый старт

### 1. Создать `.env`

```bash
cp .env.example .env
```

Минимум проверь эти переменные:
- `PG_URL`
- `AWS_S3_BUCKET`
- `AWS_S3_ENDPOINT`
- `AVATAR_BASE_URL`
- `INCIDENT_MEDIA_BASE_URL`
- `SMTP_MAIL`
- `SMTP_CODE`

`SMTP_MAIL` и `SMTP_CODE` нужны для:
- `POST /v1/users/email-code/send`
- `POST /v1/users/password-reset/send-code`
- `POST /v1/incidents/{id}/document/email`

### 2. Поднять стек

```bash
make compose-up
```

Будут запущены:
- `db` - PostgreSQL/PostGIS
- `migrator` - применение миграций
- `db-seed` - начальные геоданные по Самаре
- `minio` - S3-совместимое object storage
- `mc-init` - создание bucket и открытие download policy
- `app` - HTTP API

### 3. Полезные URL

- API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Healthcheck: `http://localhost:8080/healthz`
- Metrics: `http://localhost:8080/metrics`
- MinIO API: `http://localhost:9000`
- MinIO Console: `http://localhost:9001`

### 4. Остановить стек

```bash
make compose-down
```

## Переменные окружения

Основные переменные из `.env.example`:

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
```

Примечания:
- по умолчанию аватары и фото инцидентов лежат в одном S3 bucket;
- ключи фото инцидентов имеют вид `incidents/<incident_id>/<timestamp>.<ext>`;
- публичные URL собираются из `AVATAR_BASE_URL` и `INCIDENT_MEDIA_BASE_URL`.

## HTTP API

Базовый префикс: `/v1`

### Служебные маршруты

- `GET /healthz`
- `GET /metrics`
- `GET /swagger/index.html`

### Auth

- `POST /v1/auth/login`
- `POST /v1/auth/register`

### Users

- `POST /v1/users` - только admin
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
- `POST /v1/categories` - только admin
- `PATCH /v1/categories/{id}` - только admin
- `DELETE /v1/categories/{id}` - только admin

### Maps

- `GET /v1/maps/reverse`
- `GET /v1/maps/search`
- `POST /v1/maps/reload-cities` - только admin

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

## Модель доступа

Временные заголовки:

- `X-User-Role: admin` - для admin-only ручек
- `X-User-ID: <id>` - для привязки инцидента к текущему пользователю и маршрутов вида "мои инциденты"

### Где нужен `X-User-ID`

- `POST /v1/incidents`
- `GET /v1/my/incidents`
- `PATCH /v1/incidents/{id}`
- `DELETE /v1/incidents/{id}`
- `POST /v1/incidents/{id}/photos`
- `DELETE /v1/incidents/{id}/photos/{photoId}`
- `GET /v1/incidents/{id}/document/download`
- `GET /v1/incidents/{id}/document/print`
- `POST /v1/incidents/{id}/document/email`

### Где нужен `X-User-Role: admin`

- `POST /v1/categories`
- `PATCH /v1/categories/{id}`
- `DELETE /v1/categories/{id}`
- `POST /v1/maps/reload-cities`

## Flow по инцидентам

### Адресная модель

Для MVP инцидент хранит сразу:
- координаты: `latitude`, `longitude`;
- адресные поля: `city`, `street`, `house`, `address_text`.

Это позволяет не вводить отдельную таблицу адресов и при этом сохранить удобство для карты, фильтрации и генерации документа.

### Статусы инцидента

- `draft`
- `published`

Поведение:
- `GET /v1/incidents` возвращает только **published**;
- `GET /v1/my/incidents` возвращает инциденты текущего пользователя, включая черновики;
- draft доступен только автору или администратору.

### Фото инцидента

Загрузка идёт через `multipart/form-data` с повторяемым полем `photos`.

Ограничения в хендлере:
- максимум 5 MB на файл;
- допустимые расширения: `.png`, `.jpg`, `.jpeg`.

### Действия с документом обращения

Документ обращения трактуется как серверно-сгенерированное представление, построенное из данных инцидента.

Доступные действия:
- `GET /v1/incidents/{id}/document/download` - HTML с `Content-Disposition: attachment`
- `GET /v1/incidents/{id}/document/print` - HTML с `Content-Disposition: inline`
- `POST /v1/incidents/{id}/document/email` - отправка этого документа по email

## Команды для разработки

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

Swagger существует в двух формах:
- аннотации рядом с хендлерами;
- сгенерированные файлы в `docs/`.

Если аннотации были изменены, перегенерация делается так:

```bash
make swag-v1
```

## Примечания

- README синхронизирован с текущим кодом и миграциями.
- Если README и сгенерированный Swagger расходятся, source of truth — **код + миграции + swagger-аннотации в хендлерах**.
- `README.md` содержит тот же обзор на английском.
