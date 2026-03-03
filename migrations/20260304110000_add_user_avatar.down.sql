-- Drop avatar column from users table
ALTER TABLE users
    DROP COLUMN IF EXISTS avatar;

