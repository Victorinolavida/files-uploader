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



build: build-api #build-woker
    @echo building apps

build-api:
    @docker build --build-arg TARGET=cmd/api  -t files-uploader-api:dev .

build-woker:
    @docker build --build-arg TARGET=cmd/worker -t files-uploader-worker:dev .

