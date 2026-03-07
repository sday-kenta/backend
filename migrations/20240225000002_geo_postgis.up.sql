-- 1. Включаем расширение PostGIS в нашей базе данных
CREATE EXTENSION IF NOT EXISTS postgis;

-- 2. Создаем таблицу для кэширования ответов от Nominatim (Маппинг полей из п. 10.3 ТЗ)
CREATE TABLE IF NOT EXISTS addresses (
    id SERIAL PRIMARY KEY,
    lat DOUBLE PRECISION NOT NULL, -- Широта
    lon DOUBLE PRECISION NOT NULL, -- Долгота
    city VARCHAR(255),             -- Город
    road VARCHAR(255),             -- Улица
    house_number VARCHAR(50),      -- Номер дома
    full_address TEXT,             -- Полный адрес строкой (для отображения)
    
    -- Специальная колонка PostGIS для хранения точки (Point) в системе координат 4326 (GPS)
    geom GEOMETRY(Point, 4326)
);

-- 3. Создаем специальный пространственный индекс (GiST) для мгновенного поиска по карте
CREATE INDEX IF NOT EXISTS addresses_geom_idx ON addresses USING GIST (geom);