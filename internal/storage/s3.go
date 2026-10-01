package storage

import (
	"context"
	"files-uploader/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewObjectStorate(ctx context.Context, cfg aws.Config, appConfig config.BucketConfig) (*s3.Client, error) {
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(appConfig.URL)
		o.UsePathStyle = true
	})
	_, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	return client, nil
}
