package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrUnauthenticated is returned when a token is missing, malformed or expired.
var ErrUnauthenticated = errors.New("unauthenticated")

// Claims is the identity carried by the token.
type Claims struct {
	UserID string
}

type TokenService struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
	now      func() time.Time // injectable for tests
}

type tokenClaims struct {
	jwt.RegisteredClaims
}

type Config struct {
	Secret   string
	Issuer   string
	Audience string
	TTL      time.Duration
}

func NewService(cfg Config) (*TokenService, error) {
	switch {
	case len(cfg.Secret) < 32:
		return nil, errors.New("jwt: secret must be at least 32 bytes")
	case cfg.Issuer == "" || cfg.Audience == "":
		return nil, errors.New("jwt: issuer and audience are required")
	case cfg.TTL <= 0:
		return nil, errors.New("jwt: ttl must be positive")
	}
	return &TokenService{
		secret:   []byte(cfg.Secret),
		issuer:   cfg.Issuer,
		audience: cfg.Audience,
		ttl:      cfg.TTL,
		now:      time.Now,
	}, nil
}

func (serv *TokenService) Issue(_ context.Context, p Claims) (string, error) {
	jti := uuid.New()
	now := serv.now()
	exp := now.Add(serv.ttl)

	c := tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti.String(),
			Subject:   p.UserID,
			Issuer:    serv.issuer,
			Audience:  jwt.ClaimStrings{serv.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(serv.secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

func (serv *TokenService) Verify(_ context.Context, raw string) (Claims, error) {
	c := &tokenClaims{}
	_, err := jwt.ParseWithClaims(raw, c,
		func(*jwt.Token) (any, error) { return serv.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(serv.issuer),
		jwt.WithAudience(serv.audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(5*time.Second),
		jwt.WithTimeFunc(serv.now),
	)
	if err != nil || c.Subject == "" {
		return Claims{}, ErrUnauthenticated
	}

	return Claims{UserID: c.Subject}, nil
}
