CREATE TABLE IF NOT EXISTS roles (
    id   INT PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO roles (id, name)
VALUES
    (1, 'user'),
    (2, 'admin'),
    (3, 'premium')
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS users (
    id             SERIAL PRIMARY KEY,
    login          VARCHAR(50)  NOT NULL UNIQUE,
    email          VARCHAR(255) NOT NULL UNIQUE,
    email_verified BOOLEAN      NOT NULL DEFAULT FALSE,
    password_hash  VARCHAR(255) NOT NULL,

    last_name      VARCHAR(100) NOT NULL,
    first_name     VARCHAR(100) NOT NULL,
    middle_name    VARCHAR(100),

    phone          VARCHAR(20)  NOT NULL,
    city           VARCHAR(100) NOT NULL,
    street         VARCHAR(255) NOT NULL,
    house          VARCHAR(50)  NOT NULL,
    apartment      VARCHAR(50),
    avatar_url     TEXT,

    is_blocked     BOOLEAN      NOT NULL DEFAULT FALSE,
    role_id        INT          NOT NULL DEFAULT 1 REFERENCES roles(id),

    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT users_phone_unique UNIQUE (phone)
);
