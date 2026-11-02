package middleware

import (
	"context"
	"net/http"

	"files-uploader/internal/cookie"
	"files-uploader/internal/csrf"
	"files-uploader/internal/jwt"
	"files-uploader/pkg/http/response"

	"github.com/google/uuid"
)

type contextKey int

const userIDKey contextKey = iota

// UserID returns the authenticated user id stored by RequireAuth.
func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

func authenticate(r *http.Request, verifier jwt.TokenVerifier, cookieName string) (uuid.UUID, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return uuid.Nil, false
	}

	claims, err := verifier.Verify(r.Context(), c.Value)
	if err != nil {
		return uuid.Nil, false
	}

	id, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, false
	}

	return id, true
}

// RequireAuth verifies the session cookie and stores the user id in the
// request context. Failures respond with JSON 401.
func RequireAuth(verifier jwt.TokenVerifier, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := authenticate(r, verifier, cookieName)
			if !ok {
				response.SendError(w, "unauthenticated", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePageAuth is like RequireAuth but redirects unauthenticated requests
// to an HTML page instead of returning JSON.
func RequirePageAuth(verifier jwt.TokenVerifier, cookieName, redirectTo string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := authenticate(r, verifier, cookieName)
			if !ok {
				http.Redirect(w, r, redirectTo, http.StatusSeeOther)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireCSRF validates the double-submit CSRF token on unsafe requests.
func RequireCSRF(conf cookie.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
			default:
				if err := csrf.Verify(r, conf.CSRFName, conf.Secret); err != nil {
					response.SendError(w, "invalid csrf token", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Protect combines authentication and CSRF validation.
func Protect(verifier jwt.TokenVerifier, conf cookie.Config) func(http.Handler) http.Handler {
	auth := RequireAuth(verifier, conf.Name)
	csrfMW := RequireCSRF(conf)
	return func(next http.Handler) http.Handler {
		return auth(csrfMW(next))
	}
}
