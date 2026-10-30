package config

import (
	"errors"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	DatabaseURL string
	Port        int
	RedisAddr   string
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

	redisAddr := net.JoinHostPort(redistHost, redistPort)

	return &Config{
		AppEnv:       appEnv,
		DatabaseURL:  databaseURL,
		Port:         portInt,
		BucketConfig: bucketConfig,
		RedisAddr:    redisAddr,
	}, nil
}
