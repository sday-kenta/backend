UPDATE incidents
SET status = 'draft',
    published_at = NULL
WHERE status = 'review';

ALTER TABLE incidents
    ALTER COLUMN status SET DEFAULT 'draft';

ALTER TABLE incidents
    DROP CONSTRAINT IF EXISTS incidents_status_check;

ALTER TABLE incidents
    ADD CONSTRAINT incidents_status_check
        CHECK (status IN ('draft', 'published'));
