package http

import (
	"encoding/json"
	"files-uploader/internal/models"
	"files-uploader/internal/service"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
)

type Router struct {
	render func(http.ResponseWriter, any)
	serv   service.FileService
}

const (
	MaxSize = 1 << 32 // 4,
)

var AllowedContentTypes = []string{
	"image/jpeg",
	"image/png",
	"image/gif",
	"image/webp",
	"image/svg+xml",
	"application/pdf",
	"application/json",
	"text/csv",
	"text/plain",
	"text/html",
	"text/css",
	"text/javascript",
	"application/xml",
	"application/zip",
	"audio/mpeg",
	"audio/wav",
	"video/mp4",
	"video/webm",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

func NewRouter(render func(http.ResponseWriter, any), serv service.FileService) http.Handler {
	r := &Router{render, serv}

	mux := http.NewServeMux()
	mux.HandleFunc("/home", r.Home)
	mux.HandleFunc("POST /uploads", r.Uploads)
	return mux
}

func (r *Router) Home(w http.ResponseWriter, _ *http.Request) {
	data := map[string]any{
		"Title": "Anasayfa",
		"User":  "Uygar",
		"Year":  time.Now().Year(),
	}
	r.render(w, data)
}

type FileMetadata struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBites   int    `json:"size_bytes"`
}

type FileUploadResponse struct {
	UploadID  string         `json:"upload_id"`
	URL       string         `json:"url"`
	Method    string         `json:"method"`
	Headers   map[string]any `json:"headers"`
	ExpiresAt string         `json:"expires_at"`
}

var HardcodedUser = uuid.MustParse("a5f18e36-d915-43cb-935d-cfdbcd37e62b")

func (r *Router) Uploads(w http.ResponseWriter, re *http.Request) {

	fileMetadata := new(FileMetadata)

	decoder := json.NewDecoder(re.Body)
	defer re.Body.Close()

	err := decoder.Decode(fileMetadata)
	if err != nil {
		slog.Error("error decoding body", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if fileMetadata.FileName == "" {
		slog.Warn("file name is empty")
		http.Error(w, "file name is empty", http.StatusBadRequest)
		return
	}

	if fileMetadata.SizeBites <= 0 || fileMetadata.SizeBites > MaxSize {
		slog.Warn("file size is invalid")
		http.Error(w, "file size is invalid, have size 0 or pass 4GB", http.StatusBadRequest)
		return
	}

	if !slices.Contains(AllowedContentTypes, fileMetadata.ContentType) {
		msg := fmt.Sprintf("invalid content_type: %s", fileMetadata.ContentType)
		slog.Warn("content_type not allowed", slog.Any("error", msg))
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	file := models.NewFile(
		HardcodedUser,
		fileMetadata.FileName,
		fileMetadata.ContentType,
		fileMetadata.SizeBites,
	)
	err = r.serv.SaveFileMetadata(re.Context(), file)
	if err != nil {
		slog.Error("saving file", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	slog.Info("success save metadata")

	request, ttl, err := r.serv.PresignSingle(
		re.Context(), file.ObjectKey, file.ContentType, file.SizeBites,
	)
	if err != nil {
		slog.Error("presigning file", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	slog.Info("success presigning file")
	response := FileUploadResponse{
		UploadID: file.ID.String(),
		URL:      request.URL,
		Method:   request.Method,
		Headers: map[string]any{
			"Content-Type": file.ContentType,
		},
		ExpiresAt: ttl.String(),
	}
	resBytes, err := json.Marshal(response)
	if err != nil {
		slog.Error("marshaling file", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(resBytes)
}
