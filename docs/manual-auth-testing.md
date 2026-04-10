# Ручное тестирование: авторизация, регистрация, отправка кода

Документ покрывает **только 3 ручки**, которые реально используются во фронтенде (`AuthPanel`):

- `POST /v1/auth/login`
- `POST /v1/auth/register`
- `POST /v1/users/email-code/send`

## База

- Base URL локально: `http://localhost:8080` (если у вас другой порт/API gateway — подставьте свой).
- Заголовок для JSON: `Content-Type: application/json`

---

## 1) Логин пользователя

- **Endpoint:** `POST /v1/auth/login`
- **Назначение:** вход по `login` / `email` / `phone` + `password`

### Тело запроса

```json
{
  "identifier": "user123",
  "password": "qwerty123"
}
```

### Пример curl

```bash
curl -i -X POST "http://localhost:8080/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"identifier\":\"user123\",\"password\":\"qwerty123\"}"
```

### Ожидаемо

- `200` + JSON пользователя при корректных данных
- `400` при невалидном body
- `401` при неверном логине/пароле
- `403` если пользователь заблокирован

---

## 2) Регистрация пользователя

- **Endpoint:** `POST /v1/auth/register`
- **Назначение:** публичная регистрация пользователя

### Тело запроса (минимально валидный пример)

```json
{
  "login": "user123",
  "email": "user@example.com",
  "password": "qwerty123",
  "last_name": "Иванов",
  "first_name": "Иван",
  "middle_name": "Иванович",
  "phone": "+7 (777) 777-77-77",
  "city": "Москва",
  "street": "Тверская",
  "house": "10",
  "apartment": "15"
}
```

### Пример curl

```bash
curl -i -X POST "http://localhost:8080/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"login\":\"user123\",\"email\":\"user@example.com\",\"password\":\"qwerty123\",\"last_name\":\"Иванов\",\"first_name\":\"Иван\",\"middle_name\":\"Иванович\",\"phone\":\"+7 (777) 777-77-77\",\"city\":\"Москва\",\"street\":\"Тверская\",\"house\":\"10\",\"apartment\":\"15\"}"
```

### Ожидаемо

- `201` + JSON созданного пользователя
- `400` при ошибке валидации
- `409` если `login/email/phone` уже заняты

### Негативные проверки (обязательные)

- **Пароль < 6 символов** → `400` (правило `min=6` для `password`)
- **Пустые обязательные поля** (`login`, `email`, `first_name` и т.д.) → `400`
- **Невалидный email** → `400`
- **Телефон не по правилу** (после нормализации не 11 цифр/не начинается с `7`) → `400`

---

## 3) Отправка email-кода

- **Endpoint:** `POST /v1/users/email-code/send`
- **Назначение:** отправка кода подтверждения на email (для регистрации/смены email)

### Тело запроса

```json
{
  "email": "user@example.com",
  "purpose": "register"
}
```

`purpose` допускает: `register`, `change_email`.

### Пример curl

```bash
curl -i -X POST "http://localhost:8080/v1/users/email-code/send" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"user@example.com\",\"purpose\":\"register\"}"
```

### Ожидаемо

- `200` (или `204`, в зависимости от текущей реализации) при успешной отправке
- `400` при невалидном email/purpose/body

---

## Рекомендуемый порядок smoke-теста

1. Зарегистрировать нового пользователя через `POST /v1/auth/register`.
2. Вызвать `POST /v1/users/email-code/send` для этого email с `purpose=register`.
3. Выполнить вход этим пользователем через `POST /v1/auth/login`.

Если все 3 шага прошли — базовый auth flow работает.
