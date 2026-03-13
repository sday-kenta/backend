-- Drop avatar_url column
ALTER TABLE users
    DROP COLUMN IF EXISTS avatar_url;

