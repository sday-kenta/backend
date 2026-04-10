# backend

Объединённый Go/Fiber backend для проекта **«Сознательный гражданин»**.

Сервис уже включает users, categories, maps/geocoding, incidents, отправку обратной связи по email, Swagger, Prometheus metrics, загрузку файлов в MinIO/S3, PostgreSQL/PostGIS и доставку push-уведомлений через Firebase Cloud Messaging.

## Текущее состояние

Сейчас реализовано:
- Bearer JWT-авторизация с публичными `POST /v1/auth/register` и `POST /v1/auth/login`;
- опциональный admin bootstrap через переменные окружения;
- управление пользователями, редактирование профиля и загрузка аватара;
- отправка и проверка email-кода;
- сброс пароля по email-коду;
- admin-only создание пользователя без email confirmation, с `role` и `is_blocked`;
- CRUD для рубрикатора плюс загрузка и удаление иконок категорий;
- поиск адресов и reverse geocoding через OSM/Nominatim + PostGIS-зоны;
- CRUD для инцидентов с moderation flow `draft / review / published`;
- загрузка фотографий инцидента с проверкой прав;
- генерация HTML-документа обращения для скачивания / печати / отправки на email;
- публичная ручка обратной связи;
- регистрация push-устройств и отправка FCM-уведомлений;
- Swagger и Prometheus integration.

Важные ограничения текущего MVP:
- доставка push синхронная и best-effort; очереди, RMQ и Redis не используются;
- авторизация сейчас построена на одном access JWT, без refresh token;
- для PWA push на фронте всё равно нужны Firebase Web SDK, VAPID key и service worker;
- Firebase service account credentials должны передаваться локально или через deployment secrets и не должны попадать в git.

## Стек

- Go 1.25
- Fiber
- PostgreSQL + PostGIS
- Swagger / Swaggo
- MinIO через S3 API
- OpenStreetMap / Nominatim
- Firebase Cloud Messaging
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
- `pkg/authjwt` - генерация и разбор JWT
- `pkg/mailsender` - helpers для отправки email
- `pkg/objectstorage` - общий helper для MinIO/S3
- `pkg/pushclient` - FCM HTTP v1 sender

## Быстрый старт

### 1. Настроить `.env`

```bash
cp .env.example .env
```

Минимум проверь эти переменные:
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

Если хочешь, чтобы приложение само создало первого администратора при запуске, дополнительно настрой:
- `ADMIN_BOOTSTRAP_ENABLED=true`
- `ADMIN_BOOTSTRAP_LOGIN`
- `ADMIN_BOOTSTRAP_EMAIL`
- `ADMIN_BOOTSTRAP_PASSWORD`
- `ADMIN_BOOTSTRAP_PHONE`

Для локального тестирования push положи Firebase service account JSON по пути из `FCM_CREDENTIALS_FILE` (по умолчанию `./firebase-service-account.json`) и не коммить его. В production лучше монтировать его как secret-файл и передавать путь через `FCM_CREDENTIALS_FILE`.

`SMTP_MAIL` и `SMTP_CODE` нужны для:
- `POST /v1/users/email-code/send`
- `POST /v1/users/password-reset/send-code`
- `POST /v1/users/password-reset/reset`
- `POST /v1/incidents/{id}/document/email`
- `POST /v1/feedback`

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

Примечания:
- по умолчанию аватары, фото инцидентов и иконки категорий могут лежать в одном S3 bucket;
- ключи фото инцидентов имеют вид `incidents/<incident_id>/<timestamp>.<ext>`;
- публичные URL собираются из `AVATAR_BASE_URL`, `INCIDENT_MEDIA_BASE_URL` и `CATEGORY_MEDIA_BASE_URL`;
- при `FCM_ENABLED=false` ручки регистрации push-устройств продолжают работать на уровне хранения токенов, но фактическая доставка уведомлений выключена.

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

- `POST /v1/users` - только admin, создаёт пользователя сразу без email confirmation
- `GET /v1/users` - только admin
- `GET /v1/users/{id}` - self или admin
- `PUT /v1/users/{id}` - self или admin
- `DELETE /v1/users/{id}` - только admin
- `POST /v1/users/{id}/avatar` - self или admin
- `POST /v1/users/email-code/send`
- `POST /v1/users/email-code/verify`
- `POST /v1/users/password-reset/send-code`
- `POST /v1/users/password-reset/reset`

### Categories

- `GET /v1/categories`
- `GET /v1/categories/{id}`
- `POST /v1/categories` - только admin
- `PATCH /v1/categories/{id}` - только admin
- `POST /v1/categories/{id}/icon` - только admin
- `DELETE /v1/categories/{id}/icon` - только admin
- `DELETE /v1/categories/{id}` - только admin

### Maps

- `GET /v1/maps/reverse`
- `GET /v1/maps/search`
- `POST /v1/maps/reload-cities` - только admin

### Incidents

- `POST /v1/incidents` - нужен auth
- `GET /v1/incidents`
- `GET /v1/incidents/{id}`
- `PATCH /v1/incidents/{id}` - нужен auth
- `DELETE /v1/incidents/{id}` - нужен auth
- `GET /v1/my/incidents` - нужен auth
- `POST /v1/incidents/{id}/photos` - нужен auth
- `DELETE /v1/incidents/{id}/photos/{photoId}` - нужен auth
- `GET /v1/incidents/{id}/document/download` - нужен auth
- `GET /v1/incidents/{id}/document/print` - нужен auth
- `POST /v1/incidents/{id}/document/email` - нужен auth

### Push

- `POST /v1/push/devices` - нужен auth
- `DELETE /v1/push/devices/{deviceId}` - нужен auth

### Feedback

- `POST /v1/feedback`

## Модель доступа

Для группы `/v1` включён optional Bearer JWT parsing. Публичные маршруты можно вызывать без `Authorization`, а для защищённых нужен заголовок:

```http
Authorization: Bearer <access-token>
```

Основные правила:
- `POST /v1/auth/register` - публичный flow саморегистрации с подтверждением email;
- `POST /v1/users` - отдельный admin-only flow для мгновенного создания пользователя;
- `GET /v1/incidents` публичен, но ответ зависит от роли, если пришёл валидный JWT;
- `GET /v1/incidents/{id}` публичен только для `published`; `draft` и `review` доступны автору или администратору;
- `GET /v1/users/{id}`, `PUT /v1/users/{id}` и `POST /v1/users/{id}/avatar` доступны только self или admin;
- admin-only доступ теперь определяется по роли в JWT, а не по кастомным заголовкам.

## Flow по инцидентам

### Адресная модель

Для MVP инцидент хранит сразу:
- координаты: `latitude`, `longitude`;
- адресные поля: `city`, `street`, `house`, `address_text`.

Это позволяет не вводить отдельную таблицу адресов и при этом сохранить удобство для карты, фильтрации и генерации документа.

### Статусы инцидента

- `draft`
- `review`
- `published`

Поведение:
- если в `POST /v1/incidents` статус не передан, бэкенд сохраняет `review`;
- `GET /v1/incidents` возвращает только `published` для обычного пользователя и анонимного запроса;
- для администратора `GET /v1/incidents` по умолчанию возвращает `published + review`;
- администратор может повторять query-параметр `status`, например `?status=review&status=published`, чтобы управлять выдачей по статусам;
- `GET /v1/incidents` и `GET /v1/my/incidents` также поддерживают `category_id`;
- `GET /v1/my/incidents` возвращает инциденты текущего пользователя и поддерживает `status=draft|review|published|all`;
- `draft` и `review` доступны только автору или администратору;
- если обычный пользователь передает `status=published` в `POST /v1/incidents` или `PATCH /v1/incidents/{id}`, бэкенд сохраняет `review`;
- администратор может сохранить `published` напрямую;
- администратор не может редактировать чужой `draft` через `PATCH /v1/incidents/{id}`;
- загружать фото может только автор инцидента; при этом администратор по-прежнему может удалять чужой инцидент и его фотографии.

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

## Push-уведомления

Push-доставка реализована через FCM HTTP v1.

Поведение бэкенда:
- `POST /v1/push/devices` регистрирует или обновляет device token для авторизованного пользователя;
- `DELETE /v1/push/devices/{deviceId}` отвязывает client-generated device ID;
- поддерживаются платформы `android`, `ios` и `web`; значение `pwa` принимается и нормализуется в `web`;
- push отправляется при смене статуса инцидента:
  - любое состояние, кроме `published`, -> `published`
  - `review` -> `draft`, если действие выполнил не автор инцидента

Для локальной разработки:
- положи Firebase service account JSON по пути `FCM_CREDENTIALS_FILE`;
- не коммить этот файл;
- включи отправку через `FCM_ENABLED=true`.

Для PWA-фронта:
- используй Firebase Web SDK;
- получай web push token через VAPID key и service worker;
- отправляй этот token в `POST /v1/push/devices` с `platform=web`.

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
