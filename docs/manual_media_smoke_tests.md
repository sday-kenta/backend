# Manual smoke tests for media URL handling

## 1. Unit tests for URL helpers

Run:

```bash
go test ./internal/controller/restapi/v1 -run 'Test(BuildObjectURL|ObjectKeyFromStoredURL|ObjectURLRoundTrip)'
```

Expected:
- tests pass;
- `buildObjectURL` still returns raw key when base URL is empty;
- `buildObjectURL` still joins base URL and object key with one slash;
- `objectKeyFromStoredURL` still extracts object key from stored public URL;
- round-trip `key -> public URL -> key` works for users, incidents, categories.

## 2. Upload user avatar still stores a valid public URL

Preconditions:
- app is running;
- MinIO/S3 is configured;
- `AVATAR_BASE_URL` is set;
- user with ID `1` exists.

Request:

```bash
curl -X POST \
  'http://localhost:8080/v1/users/1/avatar' \
  -H 'accept: application/json' \
  -F 'avatar=@./avatar.png;type=image/png'
```

Expected:
- HTTP `200`;
- response contains non-empty `avatar_url`;
- `avatar_url` starts with `AVATAR_BASE_URL` and ends with `users/...png`.

## 3. Upload incident photos still stores valid public URLs

Preconditions:
- app is running;
- MinIO/S3 is configured;
- incident with ID `1` exists;
- requester user exists and has access to that incident;
- incident media base URL is configured.

Request:

```bash
curl -X POST \
  'http://localhost:8080/v1/incidents/1/photos' \
  -H 'accept: application/json' \
  -H 'X-User-ID: 1' \
  -H 'Content-Type: multipart/form-data' \
  -F 'photos=@./incident-photo.png;type=image/png'
```

Expected:
- HTTP `201`;
- response array contains at least one photo;
- every `file_url` is non-empty;
- every `file_url` starts with configured incident media base URL and contains `incidents/1/`.

## 4. Replace category icon deletes old object and saves new URL

Preconditions:
- app is running;
- MinIO/S3 is configured;
- category with ID `1` exists;
- caller is admin.

First upload:

```bash
curl -X POST \
  'http://localhost:8080/v1/categories/1/icon' \
  -H 'accept: application/json' \
  -H 'X-User-Role: admin' \
  -H 'Content-Type: multipart/form-data' \
  -F 'icon=@./icon-v1.png;type=image/png'
```

Second upload with another file:

```bash
curl -X POST \
  'http://localhost:8080/v1/categories/1/icon' \
  -H 'accept: application/json' \
  -H 'X-User-Role: admin' \
  -H 'Content-Type: multipart/form-data' \
  -F 'icon=@./icon-v2.png;type=image/png'
```

Expected:
- both requests return HTTP `200`;
- second response contains updated `icon_url`;
- `icon_url` points to the new object key;
- old icon object is deleted from MinIO/S3 best-effort.

## 5. Delete category icon still clears icon_url

Request:

```bash
curl -X DELETE \
  'http://localhost:8080/v1/categories/1/icon' \
  -H 'accept: */*' \
  -H 'X-User-Role: admin'
```

Expected:
- HTTP `204`;
- subsequent `GET /v1/categories/1` returns category with empty `icon_url`;
- old icon object is deleted from MinIO/S3 best-effort.

## 6. Regression check for empty base URL

Preconditions:
- unset media base URL for the corresponding flow.

Expected:
- uploaded media is still persisted;
- stored URL field falls back to raw object key instead of malformed URL;
- deletion based on stored raw key still works for categories.
