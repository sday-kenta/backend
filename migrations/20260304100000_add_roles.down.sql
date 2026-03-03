ALTER TABLE users ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE users SET is_admin = true WHERE role_id = (SELECT id FROM roles WHERE name = 'admin');
UPDATE users SET is_admin = false WHERE role_id != (SELECT id FROM roles WHERE name = 'admin');
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_role;
ALTER TABLE users DROP COLUMN role_id;
DROP TABLE IF EXISTS roles;
