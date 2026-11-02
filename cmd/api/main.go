package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"text/template"
	"time"

	"files-uploader/config"
	"files-uploader/internal/db"
	appRouter "files-uploader/internal/http/files"
	"files-uploader/internal/http/middleware"
	"files-uploader/internal/http/users"
	"files-uploader/internal/jwt"
	filesrepository "files-uploader/internal/repository/files"
	userrepository "files-uploader/internal/repository/users"
	fileservice "files-uploader/internal/service/files"
	usersservice "files-uploader/internal/service/users"
	"files-uploader/internal/storage"
	"files-uploader/pkg/logger"
)

func render(w http.ResponseWriter, data any) {
	tpl := template.Must(template.ParseFiles(
		"views/layouts/base.html",
	))
	if err := tpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderAuth(w http.ResponseWriter, page string, data any) {
	tpl := template.Must(template.ParseFiles(
		"views/layouts/auth.html",
		"views/auth/"+page+".html",
	))
	if err := tpl.ExecuteTemplate(w, "auth", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	appConfig, err := config.Load()
	if err != nil {
		log.Panicf("error loading config")
	}

	log := logger.NewLogger(appConfig.AppEnv)
	cfg, err := storage.LoadConfig(context.Background(), appConfig.AppEnv, appConfig.BucketConfig)
	if err != nil {
		log.Error("unable to load SDK config", slog.Any("error", err))
		panic(1)
	}

	objStorage, err := storage.NewObjectStorate(context.TODO(), cfg, appConfig.BucketConfig)
	if err != nil {
		log.Error("unable to connect to bucket", slog.Any("error", err))
		os.Exit(1)
	}

	db := db.NewDB()
	repo := filesrepository.NewFilesRepository(db)
	userRepo := userrepository.NewUserRepo(db)
	mux := http.NewServeMux()

	fileService := fileservice.NewFileServices(repo, objStorage, appConfig.BucketConfig)
	tokenService, err := jwt.NewService(appConfig.TokenConfig)
	if err != nil {
		log.Error("unable to create new jwt service", slog.Any("error", err))
		os.Exit(1)
	}
	userService := usersservice.NewUserService(userRepo, tokenService)
	protect := middleware.Protect(tokenService, appConfig.CookieConfig)
	pageAuth := middleware.RequirePageAuth(tokenService, appConfig.CookieConfig.Name, "/register")
	handler := appRouter.NewRouter(render, fileService, appConfig.CookieConfig.CSRFName, protect, pageAuth)
	userHandler := users.NewRouter(render, userService, appConfig.CookieConfig, protect)
	mux.Handle("/", handler)
	mux.Handle("/users/", http.StripPrefix("/users", userHandler))

	mux.HandleFunc("GET /login", func(w http.ResponseWriter, _ *http.Request) {
		renderAuth(w, "login", map[string]any{"Year": time.Now().Year()})
	})
	mux.HandleFunc("GET /register", func(w http.ResponseWriter, _ *http.Request) {
		renderAuth(w, "register", map[string]any{"Year": time.Now().Year()})
	})

	log.Info("starting server", slog.Int("port", appConfig.Port))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", appConfig.Port), mux); err != nil {
		log.Error("error starting server", slog.Any("error", err))
		panic(1)
	}
}
