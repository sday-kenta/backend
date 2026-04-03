ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- Существующие аккаунты считаем уже подтверждёнными (обратная совместимость).
UPDATE users SET email_verified = TRUE;
