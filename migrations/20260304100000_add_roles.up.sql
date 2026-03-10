-- Create roles table
CREATE TABLE IF NOT EXISTS roles (
    id   SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO roles (name) VALUES ('user'), ('admin'), ('premium')
ON CONFLICT (name) DO NOTHING;

-- Add role_id to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id INT;

-- user role gets id 1 from INSERT order
ALTER TABLE users ALTER COLUMN role_id SET DEFAULT 1;
UPDATE users SET role_id = 1 WHERE role_id IS NULL;
ALTER TABLE users ALTER COLUMN role_id SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT fk_users_role FOREIGN KEY (role_id) REFERENCES roles(id);
