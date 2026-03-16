# backend

Combined backend service for maps and users domains.

## Included domains
- users
- categories
- maps / geocoding

## Main HTTP routes
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

## Notes
- Database image is PostGIS because maps requires it.
- User avatar upload still uses S3/MinIO environment variables from the HTTP handler.
- Swagger docs were merged statically and should be regenerated with `make swag-v1` in a proper Go 1.25 environment.
