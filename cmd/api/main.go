package main

import (
	"context"
	"files-uploader/internal/db"
	routerFU "files-uploader/internal/http"
	"files-uploader/internal/repository"
	"files-uploader/internal/storage"
	"log/slog"
	"net/http"
	"os"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func render(w http.ResponseWriter, data any) {
	tpl := template.Must(template.ParseFiles(
		"views/layouts/base.html",
	))
	if err := tpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	mux := http.NewServeMux()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(log)

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				"minioadmin",
				"minioadmin",
				"",
			),
		))

	if err != nil {
		slog.Error("unable to load SDK config", slog.Any("error", err))
		panic(1)
	}

	objStorage, err := storage.NewObjectStorate(context.TODO(), cfg)
	if err != nil {
		slog.Error("unable to connect to bucket", slog.Any("error", err))
		panic(1)
	}

	presigner := s3.NewPresignClient(objStorage)
	service := storage.NewServiceStorate(presigner)

	db := db.NewDB()
	repo := repository.NewSqlRepo(db)

	handler := routerFU.NewRouter(render, repo, service)
	mux.Handle("/", handler)

	slog.Info("starting server on 3000")
	if err := http.ListenAndServe("localhost:3000", mux); err != nil {
		log.Error("error starting server", slog.Any("error", err))
		panic(1)
	}

}
