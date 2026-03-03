-- Add avatar column to users table for storing user profile pictures
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar BYTEA;

