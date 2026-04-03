ALTER TABLE incidents
    ALTER COLUMN status SET DEFAULT 'review';

ALTER TABLE incidents
    DROP CONSTRAINT IF EXISTS incidents_status_check;

ALTER TABLE incidents
    ADD CONSTRAINT incidents_status_check
        CHECK (status IN ('draft', 'review', 'published'));
