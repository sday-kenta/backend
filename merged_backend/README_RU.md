# backend

Объединённый backend-сервис для доменов maps и users.

## Что внутри
- users
- categories
- maps / geocoding

## Основные HTTP роуты
- `GET /healthz`
- `GET /metrics`
- `GET /swagger/index.html`
- `GET /v1/categories`
- `GET /v1/categories/{id}`
- `POST /v1/categories`
- `PATCH /v1/categories/{id}`
- `DELETE /v1/categories/{id}`
- `GET /v1/maps/reverse`
- `GET /v1/maps/search`
- `POST /v1/maps/reload-cities`
- `POST /v1/users`
- `GET /v1/users`
- `GET /v1/users/{id}`
- `PUT /v1/users/{id}`
- `DELETE /v1/users/{id}`
- `POST /v1/users/{id}/avatar`

## Замечания
- В качестве базы выбран PostGIS, потому что он нужен maps-части.
- Загрузка аватаров пока всё ещё читает S3/MinIO-переменные прямо из HTTP-хендлера.
- Swagger я свёл статически; потом лучше перегенерировать его через `make swag-v1` в окружении Go 1.25.
