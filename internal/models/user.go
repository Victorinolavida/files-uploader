package models

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        uuid.UUID
	Email     string
	Name      string
	CreatedAt time.Time
	Password  []byte
}

var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrInvalidName     = errors.New("name is required, and must be longer than 3 characters")
	ErrInvalidPassword = errors.New("password is required and must be longer than 4 characters")
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func ValidEmail(email string) error {
	if emailRegex.MatchString(email) {
		return nil
	}
	return ErrInvalidEmail
}

func ValidatePassword(p string) error {
	if len(p) <= 3 {
		return ErrInvalidPassword
	}
	return nil
}

func NewUser(
	email, name, plainPass string,
) (*User, error) {
	if err := ValidEmail(email); err != nil {
		return nil, err
	}
	if len(name) <= 2 {
		return nil, ErrInvalidName
	}
	if err := ValidatePassword(plainPass); err != nil {
		return nil, err
	}
	hash, err := hashPassword(plainPass)
	if err != nil {
		return nil, err
	}
	return &User{
		ID:        uuid.New(),
		Email:     email,
		Password:  hash,
		Name:      name,
		CreatedAt: time.Now(),
	}, nil
}

func hashPassword(password string) ([]byte, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return []byte(""), err
	}

	return hashedPassword, err
}

func CheckPassword(password string, hashedPassword []byte) bool {
	err := bcrypt.CompareHashAndPassword(
		hashedPassword,
		[]byte(password),
	)

	return err == nil
}
