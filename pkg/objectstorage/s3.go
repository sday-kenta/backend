package objectstorage

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client is a small wrapper over S3-compatible object storage (MinIO/S3).
type Client struct {
	bucket string
	client *s3.Client
}

// NewFromEnv initializes client from AWS_* environment variables already used by the project.
func NewFromEnv(ctx context.Context) (*Client, error) {
	bucket := os.Getenv("AWS_S3_BUCKET")
	endpoint := os.Getenv("AWS_S3_ENDPOINT")
	region := os.Getenv("AWS_REGION")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if bucket == "" || endpoint == "" || region == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("object storage is not configured")
	}

	awsCfg, err := awscfg.LoadDefaultConfig(
		ctx,
		awscfg.WithRegion(region),
		awscfg.WithCredentialsProvider(
			aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
			),
		),
		awscfg.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service, _ string, _ ...interface{}) (aws.Endpoint, error) {
					if service == s3.ServiceID {
						return aws.Endpoint{URL: endpoint, HostnameImmutable: true}, nil
					}

					return aws.Endpoint{}, &aws.EndpointNotFoundError{}
				},
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("objectstorage - NewFromEnv - LoadDefaultConfig: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = true
	})

	return &Client{bucket: bucket, client: client}, nil
}

// Upload uploads an object under the given key.
func (c *Client) Upload(ctx context.Context, key, contentType string, body io.Reader) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("objectstorage - Upload - PutObject: %w", err)
	}

	return nil
}

// Delete removes an object. Missing objects are ignored by S3-compatible backends.
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("objectstorage - Delete - DeleteObject: %w", err)
	}

	return nil
}
