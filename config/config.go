package config

import (
	"errors"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"files-uploader/internal/cookie"
	"files-uploader/internal/jwt"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv       string
	DatabaseURL  string
	Port         int
	RedisAddr    string
	TokenConfig  jwt.Config
	CookieConfig cookie.Config
	BucketConfig
}

type BucketConfig struct {
	Bucket   string
	Region   string
	Password string
	User     string
	URL      string
}

const (
	defaultPort    = 8000
	defaultBucket  = "test"
	defaultRegion  = "us-east-1"
	localUser      = "minioadmin"
	localPassword  = "minioadmin"
	localRedisHost = "localhost"
	localRedisPort = "6379"
)

// type TokenConfig struct {
// 	Secret   string        `koanf:"secret"`
// 	Issuer   string        `koanf:"issuer"`
// 	Audience string        `koanf:"audience"`
// 	TTL      time.Duration `koanf:"ttl"`
// }

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found")
	}

	appEnv := os.Getenv("APP_ENV")
	port := os.Getenv("PORT")
	databaseURL := os.Getenv("DATABASE_URL")
	bucket := os.Getenv("S3_BUCKET")
	region := os.Getenv("AWS_REGION")
	awsS3Url := os.Getenv("AWS_S3_URL")
	redistHost := os.Getenv("REDIST_HOST")
	redistPort := os.Getenv("REDIST_PORT")
	tokenSecret := os.Getenv("TOKEN_SECRET")
	tokenIssuer := os.Getenv("TOKEN_ISSUER")
	tokenAudience := os.Getenv("TOKEN_AUDIENCE")
	tokenTTLRaw := os.Getenv("TOKEN_TTL")

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, errors.New("port must be a int")
	}
	if portInt == 0 {
		portInt = defaultPort
	}
	if bucket == "" {
		slog.Warn("not bucket configured, using default bucket")
		bucket = defaultBucket
	}

	if redistHost == "" {
		redistHost = localRedisHost
	}
	if redistPort == "" {
		redistPort = localRedisPort
	}

	bucketConfig := BucketConfig{
		Bucket: bucket,
		Region: region,
	}
	if strings.ToLower(appEnv) != "production" {
		bucketConfig.User = localUser
		bucketConfig.Password = localPassword
		bucketConfig.URL = awsS3Url
	}

	tokenTTL, err := time.ParseDuration(tokenTTLRaw)
	if err != nil {
		slog.Error("error parsing token ttl, using default", slog.Any("error", err))
		tokenTTL = time.Hour
	}

	redisAddr := net.JoinHostPort(redistHost, redistPort)

	cookieName := envOrDefault("COOKIE_NAME", "session")
	csrfName := envOrDefault("COOKIE_CSRF_NAME", "csrf_token")
	cookiePath := envOrDefault("COOKIE_PATH", "/")
	cookieSameSite := envOrDefault("COOKIE_SAME_SITE", "lax")
	cookieDomain := os.Getenv("COOKIE_DOMAIN")

	cookieTTL := tokenTTL
	if raw := os.Getenv("COOKIE_TTL"); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil {
			slog.Warn("invalid COOKIE_TTL, using token ttl", slog.Any("error", err))
		} else {
			cookieTTL = ttl
		}
	}

	cookieSecure := strings.EqualFold(appEnv, "production")
	if raw := os.Getenv("COOKIE_SECURE"); raw != "" {
		cookieSecure = raw == "true" || raw == "1"
	}

	csrfSecret := os.Getenv("CSRF_SECRET")
	if csrfSecret == "" {
		csrfSecret = tokenSecret
	}

	return &Config{
		AppEnv:       appEnv,
		DatabaseURL:  databaseURL,
		Port:         portInt,
		BucketConfig: bucketConfig,
		RedisAddr:    redisAddr,
		TokenConfig: jwt.Config{
			Issuer:   tokenIssuer,
			Audience: tokenAudience,
			Secret:   tokenSecret,
			TTL:      tokenTTL,
		},
		CookieConfig: cookie.Config{
			Name:     cookieName,
			CSRFName: csrfName,
			Path:     cookiePath,
			Domain:   cookieDomain,
			Secure:   cookieSecure,
			SameSite: cookieSameSite,
			TTL:      cookieTTL,
			Secret:   csrfSecret,
		},
	}, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
