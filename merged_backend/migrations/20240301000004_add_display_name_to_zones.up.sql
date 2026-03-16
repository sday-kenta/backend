ALTER TABLE zones
    ADD COLUMN IF NOT EXISTS display_name text;

UPDATE zones
SET display_name = CASE LOWER(name)
    WHEN 'samara' THEN 'Самара'
    ELSE INITCAP(REPLACE(name, '-', ' '))
END
WHERE display_name IS NULL OR BTRIM(display_name) = '';

ALTER TABLE zones
    ALTER COLUMN display_name SET NOT NULL;
