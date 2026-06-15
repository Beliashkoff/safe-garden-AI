// Package objstore is a thin S3-compatible object storage client for presigned
// uploads and server-side reads. Works with MinIO (dev) and Yandex Object
// Storage (prod). The backend never proxies the upload PUT (CLAUDE.md #4); it
// only presigns the URL and later reads the object back to send to the model.
package objstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// deleteBatchSize is the S3/Yandex DeleteObjects per-request cap.
const deleteBatchSize = 1000

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

func (Disabled) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", ErrDisabled
}

func (Disabled) Get(context.Context, string) ([]byte, string, error) {
	return nil, "", ErrDisabled
}

func (Disabled) GetLimited(context.Context, string, int64) ([]byte, string, error) {
	return nil, "", ErrDisabled
}

func (Disabled) PutPublic(context.Context, string, string, []byte) (string, error) {
	return "", ErrDisabled
}

func (Disabled) DeletePrefix(context.Context, string) (int, error) {
	return 0, ErrDisabled
}

func (Disabled) DeleteKeys(context.Context, []string) error {
	return ErrDisabled
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
	// publicBase is the path-style URL prefix for publicly readable objects
	// (endpoint/bucket). Empty when no explicit endpoint is configured.
	publicBase string
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
	publicBase := ""
	if cfg.Endpoint != "" {
		publicBase = strings.TrimRight(cfg.Endpoint, "/") + "/" + cfg.Bucket
	}
	return &Client{
		s3:         client,
		presign:    s3.NewPresignClient(client),
		bucket:     cfg.Bucket,
		publicBase: publicBase,
	}, nil
}

// Ping verifies the configured bucket is reachable (HeadBucket). Used by the
// admin dependency health check; the caller supplies the timeout via ctx.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(c.bucket)})
	if err != nil {
		return fmt.Errorf("objstore: head bucket: %w", err)
	}
	return nil
}

// PutPublic uploads a small server-side object (admin catalog images) with a
// public-read ACL and returns its stable public URL. This is NOT the user
// media path — user uploads always go through presigned PUT (CLAUDE.md #4);
// catalog images need a permanent URL the mobile card can load without auth.
func (c *Client) PutPublic(ctx context.Context, key, contentType string, data []byte) (string, error) {
	if c.publicBase == "" {
		return "", fmt.Errorf("objstore: public URL base unknown (endpoint not configured)")
	}
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", fmt.Errorf("objstore: put public: %w", err)
	}
	return c.publicBase + "/" + key, nil
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

// PresignGet returns a presigned GET URL valid for ttl. Used to let the mobile
// client display a stored photo it no longer has locally (history after a
// reinstall / on another device) without the backend proxying the bytes.
func (c *Client) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("objstore: presign get: %w", err)
	}
	return req.URL, nil
}

// Get reads an object, returning its bytes and stored Content-Type. Refuses
// objects larger than maxObjectBytes (the image cap). For larger media (audio),
// use GetLimited with an explicit cap.
func (c *Client) Get(ctx context.Context, key string) (data []byte, contentType string, err error) {
	return c.GetLimited(ctx, key, maxObjectBytes)
}

// GetLimited reads an object, refusing bodies larger than maxBytes
// (ErrObjectTooLarge). The presigned PUT does not enforce a size policy on its
// own, so this is the server-side safety net.
func (c *Client) GetLimited(ctx context.Context, key string, maxBytes int64) (data []byte, contentType string, err error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("objstore: get object: %w", err)
	}
	defer func() { _ = out.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(out.Body, maxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("objstore: read object: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, "", ErrObjectTooLarge
	}
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	return body, contentType, nil
}

// DeletePrefix removes every object under prefix and returns how many were
// deleted. Used to wipe a user's media (u/{user_id}/) on account deletion. The
// listing is paginated and deletions are batched (deleteBatchSize per request).
func (c *Client) DeletePrefix(ctx context.Context, prefix string) (int, error) {
	pager := s3.NewListObjectsV2Paginator(c.s3, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
		Prefix: aws.String(prefix),
	})
	var keys []string
	for pager.HasMorePages() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("objstore: list %q: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
	}
	if err := c.DeleteKeys(ctx, keys); err != nil {
		return 0, err
	}
	return len(keys), nil
}

// DeleteKeys deletes the given object keys in batches. Deleting a missing key is
// not an error (idempotent), so this is safe to retry.
func (c *Client) DeleteKeys(ctx context.Context, keys []string) error {
	for start := 0; start < len(keys); start += deleteBatchSize {
		end := start + deleteBatchSize
		if end > len(keys) {
			end = len(keys)
		}
		ids := make([]types.ObjectIdentifier, 0, end-start)
		for _, k := range keys[start:end] {
			ids = append(ids, types.ObjectIdentifier{Key: aws.String(k)})
		}
		out, err := c.s3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(c.bucket),
			Delete: &types.Delete{Objects: ids, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("objstore: delete objects: %w", err)
		}
		if len(out.Errors) > 0 {
			msg := aws.ToString(out.Errors[0].Message)
			return fmt.Errorf("objstore: delete reported %d errors, first: %s", len(out.Errors), msg)
		}
	}
	return nil
}
