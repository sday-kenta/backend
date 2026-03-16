-- Drop unique constraint on users.phone
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_phone_unique;

