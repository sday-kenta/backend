-- Down migration is effectively a no-op now:
-- we always store avatars externally (S3) via avatar_url
-- and do not want to recreate the legacy binary avatar column.
ALTER TABLE users
    DROP COLUMN IF EXISTS avatar;

