CREATE TABLE IF NOT EXISTS incidents (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id INT NOT NULL REFERENCES categories(id),

    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    department_name VARCHAR(255) NOT NULL DEFAULT '',

    city VARCHAR(100),
    street VARCHAR(255),
    house VARCHAR(50),
    address_text TEXT NOT NULL,
    latitude DOUBLE PRECISION NOT NULL DEFAULT 0,
    longitude DOUBLE PRECISION NOT NULL DEFAULT 0,

    reporter_full_name TEXT NOT NULL,
    reporter_email VARCHAR(255) NOT NULL,
    reporter_phone VARCHAR(50) NOT NULL,
    reporter_address TEXT NOT NULL,

    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS incidents_user_id_idx ON incidents(user_id);
CREATE INDEX IF NOT EXISTS incidents_category_id_idx ON incidents(category_id);
CREATE INDEX IF NOT EXISTS incidents_status_idx ON incidents(status);
CREATE INDEX IF NOT EXISTS incidents_created_at_idx ON incidents(created_at DESC);

CREATE TABLE IF NOT EXISTS incident_photos (
    id BIGSERIAL PRIMARY KEY,
    incident_id BIGINT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    file_key TEXT NOT NULL,
    file_url TEXT NOT NULL,
    content_type VARCHAR(255),
    size_bytes BIGINT NOT NULL DEFAULT 0,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS incident_photos_incident_id_idx ON incident_photos(incident_id);
