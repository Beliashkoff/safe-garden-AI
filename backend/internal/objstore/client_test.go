package objstore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_RequiresBucket(t *testing.T) {
	_, err := New(Config{})
	require.Error(t, err)
}

// PresignPut is local SigV4 signing (no network), so it is fully unit-testable.
func TestPresignPut(t *testing.T) {
	c, err := New(Config{
		Endpoint:     "http://localhost:9000",
		Region:       "ru-central1",
		AccessKey:    "key",
		SecretKey:    "secret",
		Bucket:       "media",
		UsePathStyle: true,
	})
	require.NoError(t, err)

	url, headers, err := c.PresignPut(context.Background(), "u/123/img/a.jpg", "image/jpeg", 5*time.Minute)
	require.NoError(t, err)
	assert.Contains(t, url, "u/123/img/a.jpg")
	assert.Contains(t, url, "media") // bucket, path-style
	assert.Contains(t, url, "X-Amz-Signature")
	assert.Equal(t, "image/jpeg", headers["Content-Type"])
}

func TestDisabled(t *testing.T) {
	d := Disabled{}
	_, _, err := d.PresignPut(context.Background(), "k", "image/jpeg", time.Minute)
	assert.ErrorIs(t, err, ErrDisabled)
	_, _, err = d.Get(context.Background(), "k")
	assert.ErrorIs(t, err, ErrDisabled)
	_, err = d.DeletePrefix(context.Background(), "u/x/")
	assert.ErrorIs(t, err, ErrDisabled)
	err = d.DeleteKeys(context.Background(), []string{"k"})
	assert.ErrorIs(t, err, ErrDisabled)
}

// fakeHTTP routes S3 requests to canned XML so DeletePrefix/DeleteKeys logic
// (pagination + batching) is tested without a real bucket.
type fakeHTTP struct {
	listCalls    int
	deleteBodies []string
}

func (f *fakeHTTP) Do(req *http.Request) (*http.Response, error) {
	q := req.URL.Query()
	switch {
	case req.Method == http.MethodGet && q.Get("list-type") == "2":
		f.listCalls++
		return f.list(q.Get("continuation-token")), nil
	case req.Method == http.MethodPost && req.URL.RawQuery != "" && strings.Contains(req.URL.RawQuery, "delete"):
		body, _ := io.ReadAll(req.Body)
		f.deleteBodies = append(f.deleteBodies, string(body))
		return xmlResp(`<?xml version="1.0" encoding="UTF-8"?><DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></DeleteResult>`), nil
	default:
		return xmlResp(`<?xml version="1.0" encoding="UTF-8"?>`), nil
	}
}

// list returns page 1 (truncated, with a continuation token) then page 2.
func (f *fakeHTTP) list(token string) *http.Response {
	if token == "" {
		return xmlResp(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>media</Name><Prefix>u/x/</Prefix><KeyCount>1</KeyCount><MaxKeys>1000</MaxKeys>
  <IsTruncated>true</IsTruncated><NextContinuationToken>TOK2</NextContinuationToken>
  <Contents><Key>u/x/img/a.jpg</Key></Contents>
</ListBucketResult>`)
	}
	return xmlResp(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>media</Name><Prefix>u/x/</Prefix><KeyCount>1</KeyCount><MaxKeys>1000</MaxKeys>
  <IsTruncated>false</IsTruncated>
  <Contents><Key>u/x/img/b.jpg</Key></Contents>
</ListBucketResult>`)
}

func xmlResp(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/xml"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func testClient(h aws.HTTPClient) *Client {
	s3c := s3.New(s3.Options{
		Region:       "ru-central1",
		Credentials:  credentials.NewStaticCredentialsProvider("k", "s", ""),
		BaseEndpoint: aws.String("http://localhost:9000"),
		UsePathStyle: true,
		HTTPClient:   h,
	})
	return &Client{s3: s3c, bucket: "media"}
}

func TestDeletePrefix_PaginatesAndDeletes(t *testing.T) {
	h := &fakeHTTP{}
	n, err := testClient(h).DeletePrefix(context.Background(), "u/x/")
	require.NoError(t, err)
	assert.Equal(t, 2, n, "both pages' keys deleted")
	assert.Equal(t, 2, h.listCalls, "second page fetched via continuation token")
	require.Len(t, h.deleteBodies, 1)
	assert.Contains(t, h.deleteBodies[0], "u/x/img/a.jpg")
	assert.Contains(t, h.deleteBodies[0], "u/x/img/b.jpg")
}

func TestDeleteKeys_Empty(t *testing.T) {
	h := &fakeHTTP{}
	require.NoError(t, testClient(h).DeleteKeys(context.Background(), nil))
	assert.Empty(t, h.deleteBodies, "no request for an empty key set")
}

func TestDeletePrefix_NoObjects(t *testing.T) {
	n, err := testClient(emptyHTTP{}).DeletePrefix(context.Background(), "u/none/")
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

type emptyHTTP struct{}

func (emptyHTTP) Do(req *http.Request) (*http.Response, error) {
	if req.URL.Query().Get("list-type") == "2" {
		return xmlResp(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>media</Name><KeyCount>0</KeyCount><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated>
</ListBucketResult>`), nil
	}
	return nil, fmt.Errorf("unexpected request: %s %s", req.Method, req.URL)
}
