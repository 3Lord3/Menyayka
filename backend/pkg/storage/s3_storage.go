// Package storage provides an S3 client (used with a SeaweedFS backend) for item photos.
package storage

import (
	"context"
	"fmt"
	"io"

	// The SDK is published under the legacy "minio" path; it's a generic AWS S3
	// client and works against any S3-compatible endpoint (SeaweedFS, R2, AWS, ...).
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint       string // S3 in-network, e.g. seaweedfs:9000
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
	PublicEndpoint string // base for the in-browser image_url; empty = Endpoint
	// PublicUseSSL is separate from UseSSL: HTTP in-network, often HTTPS outside (Caddy).
	// nil = same as UseSSL.
	PublicUseSSL *bool
}

type Storage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

// New returns a client and creates the bucket if missing.
func New(ctx context.Context, cfg Config) (*Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 client init: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("s3 bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("s3 bucket create: %w", err)
		}
	}

	// Objects are publicly readable (img src); writes require the access/secret key.
	if err := client.SetBucketPolicy(ctx, cfg.Bucket, publicReadPolicy(cfg.Bucket)); err != nil {
		return nil, fmt.Errorf("s3 bucket policy: %w", err)
	}

	publicUseSSL := cfg.UseSSL
	if cfg.PublicUseSSL != nil {
		publicUseSSL = *cfg.PublicUseSSL
	}

	scheme := "http"
	if publicUseSSL {
		scheme = "https"
	}

	publicEndpoint := cfg.PublicEndpoint
	if publicEndpoint == "" {
		publicEndpoint = cfg.Endpoint
	}

	return &Storage{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: fmt.Sprintf("%s://%s/%s", scheme, publicEndpoint, cfg.Bucket),
	}, nil
}

// publicReadPolicy is the standard bucket policy for anonymous reads of all objects.
func publicReadPolicy(bucket string) string {
	return fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, bucket)
}

// Upload stores a file in the bucket and returns the object's public URL.
func (s *Storage) Upload(ctx context.Context, objectName string, content io.Reader, size int64, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, objectName, content, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("s3 put object: %w", err)
	}

	return fmt.Sprintf("%s/%s", s.publicBaseURL, objectName), nil
}