-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE IF NOT EXISTS auth.permissions (
    id SERIAL PRIMARY KEY,
    permission TEXT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS auth.roles (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    scope TEXT NOT NULL DEFAULT 'global'
);

CREATE TABLE IF NOT EXISTS auth.role_permissions (
    role_id INT NOT NULL REFERENCES auth.roles(id) ON DELETE CASCADE,
    permission_id INT NOT NULL REFERENCES auth.permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS auth.user_roles (
    user_id UUID NOT NULL,
    role_id INT NOT NULL REFERENCES auth.roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_auth_user_roles_user_id ON auth.user_roles(user_id);

INSERT INTO auth.permissions (permission) VALUES
    ('campaigns:read:masked'),
    ('campaigns:write:masked'),
    ('campaigns:pause')
ON CONFLICT (permission) DO NOTHING;

INSERT INTO auth.roles (name, scope) VALUES
    ('buyer', 'team'),
    ('support', 'global')
ON CONFLICT (name) DO NOTHING;

INSERT INTO auth.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM auth.roles r
CROSS JOIN auth.permissions p
WHERE r.name = 'buyer' AND p.permission IN ('campaigns:read:masked', 'campaigns:pause')
ON CONFLICT DO NOTHING;

INSERT INTO auth.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM auth.roles r
JOIN auth.permissions p ON p.permission = 'campaigns:read:masked'
WHERE r.name = 'support'
ON CONFLICT DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS auth.user_roles;
DROP TABLE IF EXISTS auth.role_permissions;
DROP TABLE IF EXISTS auth.roles;
DROP TABLE IF EXISTS auth.permissions;
DROP SCHEMA IF EXISTS auth;
-- +goose StatementEnd
