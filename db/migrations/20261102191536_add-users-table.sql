-- migrate:up
CREATE TABLE IF NOT EXISTS  users (
    id uuid NOT NULL PRIMARY KEY,
    email text not null unique,
    name text CONSTRAINT chk_name CHECK (LENGTH(name) >= 3),
    password bytea not null,
    created_at TIMESTAMPZ not null
);


-- migrate:down
DROP TABLE users;

