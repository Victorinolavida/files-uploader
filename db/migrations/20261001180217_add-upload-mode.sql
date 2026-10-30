-- migrate:up
ALTER TABLE files ADD COLUMN mode text NOT NULL default 'single';


-- migrate:down
ALTER TABLE FILES DROP COLUMN mode;

