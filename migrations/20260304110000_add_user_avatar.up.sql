-- Legacy migration was adding `avatar BYTEA` column.
-- We now store only an avatar identifier/URL in `avatar_url` (TEXT) for S3.
-- To avoid unused binary column, ensure it is removed.
ALTER TABLE users
    DROP COLUMN IF EXISTS avatar;

