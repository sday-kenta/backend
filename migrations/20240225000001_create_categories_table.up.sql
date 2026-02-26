CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    icon_url VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE
);

INSERT INTO categories (title) VALUES ('Нарушение правил парковки');
INSERT INTO categories (title) VALUES ('Продажа просроченных товаров');