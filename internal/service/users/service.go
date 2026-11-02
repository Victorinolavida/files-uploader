package usersservice

import (
	"context"
	"errors"

	"files-uploader/internal/jwt"
	"files-uploader/internal/models"
	"files-uploader/internal/repository"
	"files-uploader/internal/service"

	"github.com/google/uuid"
)

type UserServ struct {
	repo      repository.UserRepository
	tokenServ service.JwtService
}

var (
	ErrEmailAlredyRegister = errors.New("email already register")
	ErrInvalidCredentials  = errors.New("invalid credentials")
)

func NewUserService(repo repository.UserRepository, tokenServ service.JwtService) service.UserService {
	return &UserServ{repo, tokenServ}
}

func (u *UserServ) RegisterUser(ctx context.Context, user *models.User) (string, error) {
	exist, err := u.repo.ExistForEmail(ctx, user.Email)
	if err != nil {
		return "", err
	}
	if exist {
		return "", ErrEmailAlredyRegister
	}
	if err := u.repo.Save(ctx, user); err != nil {
		return "", err
	}
	token, err := u.tokenServ.Issue(ctx, jwt.Claims{
		UserID: user.ID.String(),
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func (u *UserServ) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *UserServ) LoginUser(ctx context.Context, email string, password string) (*models.User, string, error) {
	user, err := u.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}

	if !models.CheckPassword(password, user.Password) {
		return nil, "", ErrInvalidCredentials
	}

	token, err := u.tokenServ.Issue(ctx, jwt.Claims{
		UserID: user.ID.String(),
	})
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}
