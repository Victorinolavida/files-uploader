set dotenv-load := true

db-new name:
    dbmate new {{name}}

db-up:
    dbmate up

db-down:
    dbmate down

api:
    go run ./cmd/api

worker:
    go run ./cmd/worker
