-- migrate:up

ALTER TABLE files ADD COLUMN multipart_upload_id text;
ALTER TABLE files ADD COLUMN part_size int;
ALTER TABLE files ADD COLUMN part_count int default 1;

-- migrate:down

ALTER TABLE files DROP COLUMN multipart_upload_id;
ALTER TABLE files DROP COLUMN part_size;
ALTER TABLE files DROP COLUMN part_count;

