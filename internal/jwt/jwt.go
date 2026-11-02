package jwt

import (
	"context"
)

type TokenVerifier interface {
	Verify(ctx context.Context, raw string) (Claims, error)
}

type TokenIssuer interface {
	Issue(ctx context.Context, p Claims) (string, error)
}
