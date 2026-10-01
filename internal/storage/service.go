package storage

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const Bucket = "files-uploader"

type Service struct {
	client *s3.PresignClient
	bucket string
}

func NewServiceStorate(client *s3.PresignClient) *Service {
	return &Service{
		client: client,
		bucket: Bucket,
	}
}

func (s *Service) PresignSingle(ctx context.Context, key, contentType string, size int) (*v4.PresignedHTTPRequest, time.Time, error) {
	ttl := 15 * time.Minute
	req, err := s.client.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(size)),
	}, s3.WithPresignExpires(ttl))

	return req, time.Now().Add(ttl), err
}
