-- Add avatar_url column for storing avatar identifier or URL
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_url TEXT;

