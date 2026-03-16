-- Add unique constraint on users.phone to prevent duplicate phone numbers
ALTER TABLE users
    ADD CONSTRAINT users_phone_unique UNIQUE (phone);

