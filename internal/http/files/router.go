package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"files-uploader/internal/dto"
	"files-uploader/internal/models"
	"files-uploader/internal/service"
	"files-uploader/pkg/http/response"

	"github.com/google/uuid"
)

type Router struct {
	render func(http.ResponseWriter, any)
	serv   service.FileService
}

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
	"application/x-apple-diskimage",
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
	mux.HandleFunc("/", notFound)
	mux.HandleFunc("/home", r.Home)
	mux.HandleFunc("POST /uploads", r.Uploads)
	mux.HandleFunc("POST /uploads/{id}/parts", r.UploadParts)
	mux.HandleFunc("POST /uploads/{id}/complete", r.FileComplete)
	mux.HandleFunc("GET /uploads/list", r.ListUploads)
	return mux
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response.SendError(w, "not found", http.StatusNotFound)
}

func (r *Router) Home(w http.ResponseWriter, _ *http.Request) {
	data := map[string]any{
		"Year": time.Now().Year(),
	}
	r.render(w, data)
}

type FileMetadata struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBites   int    `json:"size_bytes"`
}

var HardcodedUser = uuid.MustParse("a5f18e36-d915-43cb-935d-cfdbcd37e62b")

func (r *Router) Uploads(w http.ResponseWriter, re *http.Request) {
	fileMetadata, err := response.ReadFromBody[FileMetadata](re)
	if err != nil {
		if errors.Is(err, response.ErrEmtpyBody) {
			response.SendError(w, err.Error(), http.StatusBadRequest)
		}

		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if fileMetadata.FileName == "" {
		slog.Warn("file name is empty")
		response.SendError(w, "file name is empty", http.StatusBadRequest)
		return
	}

	if fileMetadata.SizeBites <= 0 || fileMetadata.SizeBites > models.MaxSize {
		slog.Warn("file size is invalid")
		response.SendError(w, "file size is invalid, have size 0 or pass 4GB", http.StatusBadRequest)
		return
	}

	if !slices.Contains(AllowedContentTypes, fileMetadata.ContentType) {
		msg := fmt.Sprintf("invalid content_type: %s", fileMetadata.ContentType)
		slog.Warn("content_type not allowed", slog.Any("error", msg))
		response.SendError(w, msg, http.StatusBadRequest)
		return
	}

	file, err := models.NewFile(
		HardcodedUser,
		fileMetadata.FileName,
		fileMetadata.ContentType,
		fileMetadata.SizeBites,
	)
	if err != nil {
		slog.Error("invalid data", slog.Any("error", err))
		response.SendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = r.serv.SaveFileMetadata(re.Context(), file)
	if err != nil {
		slog.Error("saving file", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	slog.Info("success save metadata")

	slog.Info("handle presign file", slog.String("mode", file.Mode))
	if file.Mode == string(models.SinglePart) {
		out, ttl, err := r.serv.PresignSingle(re.Context(), file.ObjectKey, file.ContentType, file.SizeBites)
		if err != nil {
			slog.Error("presign file", slog.Any("error", err))
			response.SendError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		resp := dto.FileSingleUploadResponse{
			CommonFileResponse: dto.CommonFileResponse{
				UploadID:  file.ID.String(),
				ExpiresAt: ttl.String(),
				Mode:      file.Mode,
			},
			Method: out.Method,
			URL:    out.URL,
			Headers: map[string]any{
				"Content-Type": file.ContentType,
			},
		}
		resBytes, err := json.Marshal(resp)
		if err != nil {
			slog.Error("marshaling file", slog.Any("error", err))
			response.SendError(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Write(resBytes)
		return
	}

	out, ttl, err := r.serv.PresignMultipart(
		re.Context(), file,
	)
	if err != nil {
		slog.Error("presigning multipart file", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	resp := dto.FileMultipleUploadResponse{
		CommonFileResponse: dto.CommonFileResponse{
			ExpiresAt: ttl.String(),
			Mode:      file.Mode,
			UploadID:  *file.MultipartUploadID,
		},
		PartCount: file.PartCount,
		PartSize:  file.PartSize,
		Parts: []dto.FileMultipleUploadPart{
			{
				PartNumber: 1,
				Method:     out.Method,
				URL:        out.URL,
				Headers: map[string]any{
					"Content-Type": file.ContentType,
				},
			},
		},
	}
	resBytes, err := json.Marshal(resp)
	if err != nil {
		slog.Error("marshaling file", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Write(resBytes)
}

func (r *Router) UploadParts(w http.ResponseWriter, re *http.Request) {
	uploadID := re.PathValue("id")
	if uploadID == "" {
		response.SendError(w, "upload_id must be not empty", http.StatusBadRequest)
		return
	}

	body := new(dto.FilePartsRequest)
	decoder := json.NewDecoder(re.Body)
	err := decoder.Decode(body)
	defer re.Body.Close()
	if err != nil {
		if err == io.EOF {
			response.SendError(w, "body empty", http.StatusBadRequest)
			return
		}
		slog.Error("error decoding body", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(body.PartNumbers) == 0 {
		response.SendError(w, "part_numbers is empty", http.StatusBadRequest)
		return
	}
	if len(body.PartNumbers) > 10 {
		response.SendError(w, "part_numbers is too big", http.StatusBadRequest)
		return
	}
	seen := make(map[int]bool, len(body.PartNumbers))
	for _, n := range body.PartNumbers {
		if n < 1 || seen[n] {
			response.SendError(w, "invalid part number", http.StatusBadRequest)
			return
		}
		seen[n] = true
	}

	file, err := r.serv.FindFileByMultipartID(re.Context(), uploadID)
	if err != nil {
		response.SendError(w, "file not found", http.StatusBadRequest)
		return
	}
	if file.Mode != string(models.MultiPart) || file.MultipartUploadID == nil || file.PartCount == 0 {
		slog.Error("file is an invalid to use this endpoint", slog.Any("id", file.ID.String()))
		response.SendError(w, "file not found", http.StatusConflict)
		return
	}

	if file.Status != string(models.PendingStatus) {
		slog.Error("file is an invalid status",
			slog.Any("id", file.ID.String()),
			slog.Any("status", file.Status),
		)
		response.SendError(w, "file is already processed", http.StatusConflict)
		return
	}
	for _, n := range body.PartNumbers {
		if n > file.PartCount {
			slog.Error("invalid part number, args is bigger thatn part_count")
			response.SendError(w, "invalid part number", http.StatusBadRequest)
			return
		}
	}

	output, ttl, err := r.serv.PresignPartRange(re.Context(), file.ObjectKey, *file.MultipartUploadID, body.PartNumbers)
	if err != nil {
		slog.Error("error presigned parts", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := dto.FilePartsResponse{
		ExpiresAt: ttl.String(),
	}
	parts := make([]dto.FilePartResponse, len(output))
	for idx, o := range output {
		parts[idx] = dto.FilePartResponse{
			PartNumber: body.PartNumbers[idx],
			URL:        o.URL,
		}
	}
	resp.Parts = parts
	responseBytes, err := json.Marshal(resp)
	if err != nil {
		slog.Error("marshaling reponsee ", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func (r *Router) FileComplete(w http.ResponseWriter, re *http.Request) {
	uploadID := re.PathValue("id")
	if uploadID == "" {
		response.SendError(w, "upload_id must be not empty", http.StatusBadRequest)
		return
	}
	file, err := r.serv.FindFileByMultipartID(re.Context(), uploadID)
	if err != nil {
		response.SendError(w, "file not found", http.StatusBadRequest)
		return
	}

	if file.Mode != string(models.MultiPart) || file.MultipartUploadID == nil || file.PartCount == 0 {
		slog.Error("file is an invalid to use this endpoint", slog.Any("id", file.ID.String()))
		response.SendError(w, "file not found", http.StatusConflict)
		return
	}

	parts, err := r.serv.ListAllParts(re.Context(), file.ObjectKey, *file.MultipartUploadID)
	if err != nil {
		slog.Error("can not list parts", slog.Any("id", file.ID.String()))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(parts.Parts) != file.PartCount {
		response.SendError(w, "number of parts not match", http.StatusConflict)
		return
	}
	size := 0
	partNumbers := make([]int, 0)
	for _, part := range parts.Parts {
		size += int(*part.Size)
		partNumbers = append(partNumbers, int(*part.PartNumber))
	}
	if !isConsecutive(partNumbers) {
		response.SendError(w, "some part is missing", http.StatusConflict)
		return
	}
	if size != file.SizeBites {
		response.SendError(w, "file size and sum part file size does not match", http.StatusConflict)
		return
	}

	err = r.serv.CompleteFileUpload(re.Context(), file, parts)
	if err != nil {
		slog.Error("can not complete file upload", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	res := dto.FileCompletedResponse{
		UploadID: *file.MultipartUploadID,
		Status:   "completing",
	}

	resBytes, _ := json.Marshal(res)

	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

func (r *Router) ListUploads(w http.ResponseWriter, re *http.Request) {
	files, err := r.serv.ListFiles(re.Context(), HardcodedUser)
	if err != nil {
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := make([]dto.FileListItem, 0, len(files))
	for _, f := range files {
		presign, _, err := r.serv.PresignDownload(re.Context(), f.ObjectKey, f.FileName, f.ContentType)
		if err != nil {
			slog.Error("presign download", slog.Any("error", err))
			continue
		}
		resp = append(resp, dto.FileListItem{
			ID:          f.ID.String(),
			Name:        f.FileName,
			Size:        f.SizeBites,
			ContentType: f.ContentType,
			CreatedAt:   f.CreatedAt,
			DownloadURL: presign.URL,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	resBytes, _ := json.Marshal(resp)
	w.WriteHeader(http.StatusOK)
	w.Write(resBytes)
}

func isConsecutive(nums []int) bool {
	if len(nums) <= 1 {
		return true
	}
	seen := make(map[int]bool)
	minNum := nums[0]
	maxNum := nums[0]

	for _, num := range nums {
		if seen[num] { // duplicated
			return false
		}
		seen[num] = true
		if num < minNum {
			minNum = num
		}
		if num > maxNum {
			maxNum = num
		}

	}
	return maxNum-minNum == len(nums)-1
}
