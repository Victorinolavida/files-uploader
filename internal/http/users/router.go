package users

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"files-uploader/internal/cookie"
	"files-uploader/internal/dto"
	"files-uploader/internal/http/middleware"
	"files-uploader/internal/models"
	userrepository "files-uploader/internal/repository/users"
	"files-uploader/internal/service"
	usersservice "files-uploader/internal/service/users"
	"files-uploader/pkg/http/response"
)

type Router struct {
	render     func(http.ResponseWriter, any)
	serv       service.UserService
	cookieConf cookie.Config
}

func NewRouter(render func(http.ResponseWriter, any), serv service.UserService, cookieConf cookie.Config, protect func(http.Handler) http.Handler) http.Handler {
	r := &Router{render, serv, cookieConf}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", r.Register)
	mux.HandleFunc("POST /login", r.Login)
	mux.Handle("GET /me", protect(http.HandlerFunc(r.Me)))
	mux.Handle("POST /logout", protect(http.HandlerFunc(r.Logout)))
	return mux
}

func (r *Router) Register(w http.ResponseWriter, re *http.Request) {
	data, err := response.ReadFromBody[dto.RegisterRequest](re)
	if err != nil {
		if errors.Is(err, response.ErrEmtpyBody) {
			response.SendError(w, err.Error(), http.StatusBadRequest)
			return
		}
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	user, err := models.NewUser(data.Email, data.Name, data.Password)
	if err != nil {
		response.SendError(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := r.serv.RegisterUser(re.Context(), user)
	if err != nil {
		if errors.Is(err, usersservice.ErrEmailAlredyRegister) {
			response.SendError(w, err.Error(), http.StatusBadRequest)
			return
		}
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := cookie.Set(w, r.cookieConf, token); err != nil {
		slog.Error("error setting session cookie", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := dto.UserLogResponse{
		ID:   user.ID.String(),
		Name: user.Name,
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		slog.Error("error marshal response", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func (r *Router) Login(w http.ResponseWriter, re *http.Request) {
	req, err := response.ReadFromBody[dto.LoginRequest](re)
	if err != nil {
		if errors.Is(err, response.ErrEmtpyBody) {
			response.SendError(w, err.Error(), http.StatusBadRequest)
			return
		}
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := models.ValidEmail(req.Email); err != nil {
		response.SendError(w, err.Error(), http.StatusBadRequest)
		return
	}
	user, token, err := r.serv.LoginUser(re.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usersservice.ErrInvalidCredentials) {
			response.SendError(w, err.Error(), http.StatusBadRequest)
			return
		}

		if errors.Is(err, userrepository.ErrUserNotFound) {
			response.SendError(w, usersservice.ErrInvalidCredentials.Error(), http.StatusBadRequest)
			return
		}
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if err := cookie.Set(w, r.cookieConf, token); err != nil {
		slog.Error("error setting session cookie", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := dto.UserLogResponse{
		ID:   user.ID.String(),
		Name: user.Name,
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		slog.Error("error marshal response", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func (r *Router) Me(w http.ResponseWriter, re *http.Request) {
	userID, ok := middleware.UserID(re.Context())
	if !ok {
		response.SendError(w, "unauthenticated", http.StatusUnauthorized)
		return
	}

	user, err := r.serv.GetUser(re.Context(), userID)
	if err != nil {
		if errors.Is(err, userrepository.ErrUserNotFound) {
			response.SendError(w, "user not found", http.StatusNotFound)
			return
		}
		slog.Error("error getting user", slog.Any("error", err))
		response.SendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := dto.UserLogResponse{
		ID:   user.ID.String(),
		Name: user.Name,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("error encoding user response", slog.Any("error", err))
	}
}

func (r *Router) Logout(w http.ResponseWriter, _ *http.Request) {
	cookie.Clear(w, r.cookieConf)
	w.WriteHeader(http.StatusNoContent)
}
