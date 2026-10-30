# files-uploader

A small file upload service written in Go. It supports single-part uploads and
S3 multipart uploads (presigned URLs), stores file metadata in SQLite, and uses
MinIO event notifications delivered through Redis to process/validate uploaded
objects in a background worker.

## Technologies

- [Go](https://go.dev/) 1.26+
- [SQLite](https://www.sqlite.org/) (local metadata database)
- [MinIO](https://min.io/) (S3-compatible object storage)
- [Redis](https://redis.io/) (queue for MinIO bucket events)
- [dbmate](https://github.com/amacneil/dbmate) (database migrations)
- [Docker](https://www.docker.com/) / Docker Compose (MinIO, Redis, Redis Insight)
- [just](https://github.com/casey/just) (optional command runner)

## Running locally

1. Start the infrastructure (MinIO, Redis, and bucket initialization):

   ```sh
   docker compose up -d
   ```

   MinIO API is available at `http://localhost:9000`, the console at
   `http://localhost:9001` (user/password: `minioadmin`), and Redis at
   `localhost:6379`.

2. Create a `.env` file in the project root:

   ```env
   DATABASE_URL=sqlite:./local.db
   PORT=3000

   AWS_REGION=us-east-1
   S3_BUCKET=files-uploader
   AWS_S3_URL=http://localhost:9000
   ```

3. Apply the database migrations:

   ```sh
   dbmate up
   # or
   just db-up
   ```

4. Start the API server:

   ```sh
   go run ./cmd/api
   ```

5. In another terminal, start the worker that consumes MinIO events from Redis:

   ```sh
   go run ./cmd/worker
   ```

The API listens on `http://localhost:3000` (configurable via `PORT`).

## API

| Method | Path                      | Description                                  |
| ------ | ------------------------- | -------------------------------------------- |
| GET    | `/home`                   | Simple HTML landing page                     |
| POST   | `/uploads`                | Create an upload and get a presigned URL     |
| POST   | `/uploads/{id}/parts`     | Get presigned URLs for multipart upload parts |
| POST   | `/uploads/{id}/complete`  | Complete a multipart upload                  |

Example request for `POST /uploads`:

```sh
curl -X POST http://localhost:3000/uploads \
  -H 'Content-Type: application/json' \
  -d '{"file_name":"hello.png","content_type":"image/png","size_bytes":1024}'
```

## Useful commands

```sh
just db-new add-something   # create a new migration
just db-up                  # run pending migrations
just db-down                # roll back the last migration
```

## Notes

- `local.db` and `.env` are ignored by git.
- In non-production environments the app uses the local MinIO credentials
  (`minioadmin` / `minioadmin`) and the `AWS_S3_URL` endpoint.
