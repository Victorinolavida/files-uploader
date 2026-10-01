package storage

import (
	appCfg "files-uploader/config"
	"strings"

	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func LoadConfig(ctx context.Context, env string, appConf appCfg.BucketConfig) (aws.Config, error) {
	options := []func(*config.LoadOptions) error{
		config.WithRegion(appConf.Region),
	}
	if strings.ToLower(env) != "production" {
		options = append(options, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				appConf.User,
				appConf.Password,
				"",
			),
		))
	}

	return config.LoadDefaultConfig(ctx, options...)
}
