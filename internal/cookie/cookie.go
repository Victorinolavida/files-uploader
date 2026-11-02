package cookie

import (
	"net/http"
	"strings"
	"time"

	"files-uploader/internal/csrf"
)

type Config struct {
	Name     string
	CSRFName string
	Path     string
	Domain   string
	Secure   bool
	SameSite string
	TTL      time.Duration
	Secret   string
}

// Set writes the session cookie and the csrf cookie.
func Set(w http.ResponseWriter, conf Config, token string) error {
	csrfToken, err := csrf.Issue(conf.Secret)
	if err != nil {
		return err
	}

	maxAge := int(conf.TTL.Seconds())
	expires := time.Now().Add(conf.TTL)

	http.SetCookie(w, &http.Cookie{
		Name:     conf.Name,
		Value:    token,
		Path:     conf.Path,
		Domain:   conf.Domain,
		Secure:   conf.Secure,
		HttpOnly: true,
		SameSite: sameSite(conf.SameSite),
		MaxAge:   maxAge,
		Expires:  expires,
	})

	// Not HttpOnly on purpose: the frontend has to read it to send it back in
	// the X-CSRF-Token header.
	http.SetCookie(w, &http.Cookie{
		Name:     conf.CSRFName,
		Value:    csrfToken,
		Path:     conf.Path,
		Domain:   conf.Domain,
		Secure:   conf.Secure,
		HttpOnly: false,
		SameSite: sameSite(conf.SameSite),
		MaxAge:   maxAge,
		Expires:  expires,
	})

	return nil
}

// Clear expires the session cookie and the csrf cookie.
func Clear(w http.ResponseWriter, conf Config) {
	for _, name := range []string{conf.Name, conf.CSRFName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     conf.Path,
			Domain:   conf.Domain,
			Secure:   conf.Secure,
			HttpOnly: name == conf.Name,
			SameSite: sameSite(conf.SameSite),
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		})
	}
}

func sameSite(value string) http.SameSite {
	switch strings.ToLower(value) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
