// Package objstore is a thin S3-compatible object storage client for presigned
// uploads and server-side reads. Works with MinIO (dev) and Yandex Object
// Storage (prod). The backend never proxies the upload PUT (CLAUDE.md #4); it
// only presigns the URL and later reads the object back to send to the model.
package objstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// maxObjectBytes caps server-side reads as a safety net against an oversized
// PUT (presigned PUT does not enforce a size policy on its own). The image
// limit is 10 MB (ARCH §8.2); we allow a little slack before refusing.
const maxObjectBytes = 12 * 1024 * 1024

// ErrObjectTooLarge is returned by Get when the stored object exceeds the cap.
var ErrObjectTooLarge = errors.New("objstore: object too large")

// ErrDisabled is returned by Disabled operations when object storage is not
// configured.
var ErrDisabled = errors.New("objstore: object storage not configured")

// Disabled is a no-op object store used when S3 is unconfigured (dev without
// MinIO). Presign and Get fail with ErrDisabled, so the app still boots for
// text-only chat; photo operations degrade rather than crash.
type Disabled struct{}

func (Disabled) PresignPut(context.Context, string, string, time.Duration) (string, map[string]string, error) {
	return "", nil, ErrDisabled
}

func (Disabled) Get(context.Context, string) ([]byte, string, error) {
	return nil, "", ErrDisabled
}

// Config configures the S3-compatible client.
type Config struct {
	Endpoint     string
	Region       string
	AccessKey    string
	SecretKey    string
	Bucket       string
	UsePathStyle bool
}

// Client wraps an S3 client + presigner bound to a single bucket.
type Client struct {
	s3      *s3.Client
	presign *s3.PresignClient
	bucket  string
}

// New builds the client. The bucket is required; an empty endpoint falls back
// to AWS defaults (only meaningful in tests).
func New(cfg Config) (*Client, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("objstore: bucket is empty")
	}
	opts := s3.Options{
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		UsePathStyle: cfg.UsePathStyle,
	}
	if cfg.Endpoint != "" {
		opts.BaseEndpoint = aws.String(cfg.Endpoint)
	}
	client := s3.New(opts)
	return &Client{
		s3:      client,
		presign: s3.NewPresignClient(client),
		bucket:  cfg.Bucket,
	}, nil
}

// PresignPut returns a presigned PUT URL valid for ttl. The content type is
// signed, so the client MUST send the same Content-Type header (returned in the
// headers map) on its PUT.
func (c *Client) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (url string, headers map[string]string, err error) {
	req, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", nil, fmt.Errorf("objstore: presign put: %w", err)
	}
	return req.URL, map[string]string{"Content-Type": contentType}, nil
}

// Get reads an object, returning its bytes and stored Content-Type. Refuses
// objects larger than maxObjectBytes.
func (c *Client) Get(ctx context.Context, key string) (data []byte, contentType string, err error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("objstore: get object: %w", err)
	}
	defer func() { _ = out.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(out.Body, maxObjectBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("objstore: read object: %w", err)
	}
	if len(body) > maxObjectBytes {
		return nil, "", ErrObjectTooLarge
	}
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	return body, contentType, nil
}
