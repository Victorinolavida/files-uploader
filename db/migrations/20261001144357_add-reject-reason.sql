-- migrate:up
ALTER TABLE files ADD COLUMN reject_reason text;


-- migrate:down

ALTER TABLE FILES DROP COLUMN reject_reason;
