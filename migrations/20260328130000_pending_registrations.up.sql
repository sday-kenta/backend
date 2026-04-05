CREATE TABLE IF NOT EXISTS pending_registrations (
    email         VARCHAR(255) PRIMARY KEY,
    login         VARCHAR(50)  NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    last_name     VARCHAR(100) NOT NULL,
    first_name    VARCHAR(100) NOT NULL,
    middle_name   VARCHAR(100),
    phone         VARCHAR(20)  NOT NULL,
    city          VARCHAR(100) NOT NULL,
    street        VARCHAR(255) NOT NULL,
    house         VARCHAR(50)  NOT NULL,
    apartment     VARCHAR(50),
    role          VARCHAR(50)  NOT NULL DEFAULT 'user',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
