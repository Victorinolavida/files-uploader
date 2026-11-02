package users

import (
	"errors"
	"net/http"

	"files-uploader/internal/dto"
	"files-uploader/internal/models"
	"files-uploader/internal/service"
	"files-uploader/pkg/http/response"

	"github.com/go-jose/go-jose/jwt"
)

type Router struct {
	render    func(http.ResponseWriter, any)
	serv      service.UserService
	tokenServ service.JwtService
}

func NewRouter(render func(http.ResponseWriter, any), serv service.UserService, tokenServ service.JwtService) http.Handler {
	r := &Router{render, serv, tokenServ}

	mux := http.NewServeMux()

	mux.HandleFunc("/auth/register", r.Register)
	// mux.HandleFunc("/home", r.Home)
	// mux.HandleFunc("POST /uploads", r.Uploads)
	// mux.HandleFunc("POST /uploads/{id}/parts", r.UploadParts)
	// mux.HandleFunc("POST /uploads/{id}/complete", r.FileComplete)
	// mux.HandleFunc("GET /uploads/list", r.ListUploads)
	return mux
}

func (r *Router) Register(w http.ResponseWriter, re *http.Request) {
	data, err := response.ReadFromBody[dto.LoginRequest](re)
	if err != nil {
		if errors.Is(err, response.ErrEmtpyBody) {
			response.SendError(w, err.Error(), http.StatusBadRequest)
			return
		}
		response.SendError(w, "internal server error", http.StatusInternalServerError)
	}
	user, err := models.NewUser(data.Email, data.Name, data.Password)
	if err != nil {
		response.SendError(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = r.serv.CreateUser(re.Context(), user)
	if err == nil {
		response.SendError(w, "internal server error", http.StatusBadRequest)
		return
	}

	token, err := r.tokenServ.Issue(re.Context(), jwt.Claims{
		UserID: user.ID,
	})
}
