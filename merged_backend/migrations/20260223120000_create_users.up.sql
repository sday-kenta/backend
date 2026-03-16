CREATE TABLE IF NOT EXISTS users (
    id             SERIAL PRIMARY KEY,
    login          VARCHAR(50)  NOT NULL UNIQUE,
    email          VARCHAR(255) NOT NULL UNIQUE,
    password_hash  VARCHAR(255) NOT NULL,

    last_name      VARCHAR(100) NOT NULL,
    first_name     VARCHAR(100) NOT NULL,
    middle_name    VARCHAR(100),

    phone          VARCHAR(20)  NOT NULL,
    city           VARCHAR(100) NOT NULL,
    street         VARCHAR(255) NOT NULL,
    house          VARCHAR(50)  NOT NULL,
    apartment      VARCHAR(50),

    is_blocked     BOOLEAN      NOT NULL DEFAULT FALSE,

    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

