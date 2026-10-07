//go:build integration

// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
)

type countParts struct {
	base  http.RoundTripper
	first atomic.Int64
}

func (c *countParts) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Method == "PUT" && r.URL.Query().Get("partNumber") == "1" {
		c.first.Add(1)
	}
	return c.base.RoundTrip(r)
}
func TestMultipartResumeAndAbort(t *testing.T) {
	ctx := context.Background()
	endpoint := os.Getenv("INSONIC_FIXTURE_S3")
	if endpoint == "" {
		t.Fatal("fixture required")
	}
	client, e := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4("", "", ""), Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
	if e != nil {
		t.Fatal(e)
	}
	bucket := "recover-" + contracts.ID()
	if e = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"}); e != nil {
		t.Fatal(e)
	}
	defer client.RemoveBucket(ctx, bucket)
	counter := &countParts{base: http.DefaultTransport}
	s, e := NewS3(ctx, S3Config{Endpoint: "http://" + endpoint, Bucket: bucket, Region: "us-east-1", Authentication: "anonymous", AddressingStyle: "path"}, nil, counter)
	if e != nil {
		t.Fatal(e)
	}
	data := bytes.Repeat([]byte("stream"), 4<<20)
	hash := sha256.Sum256(data)
	p := catalog.Publication{ID: contracts.ID(), Key: "objects/" + contracts.ID(), Digest: hex.EncodeToString(hash[:]), Size: int64(len(data)), State: "pending"}
	source := bytesFile(t, data)
	stopped := errors.New("fixture stopped after first persisted part")
	p, e = s.PublishImmutable(ctx, p, source, func(next catalog.Publication) (catalog.Publication, error) {
		if len(next.Parts) == 1 {
			return next, stopped
		}
		return next, nil
	})
	if e == nil || p.UploadID == "" || len(p.Parts) != 1 {
		t.Fatalf("interruption %+v %v", p, e)
	}
	before := counter.first.Load()
	upload := p.UploadID
	p, e = s.PublishImmutable(ctx, p, source, func(next catalog.Publication) (catalog.Publication, error) { return next, nil })
	if e != nil || p.UploadID != upload || counter.first.Load() != before {
		t.Fatalf("resume failed/reuploaded accepted part: %v", e)
	}
	p.State = "retiring"
	if e = s.DeleteUnreferenced(ctx, p); e != nil {
		t.Fatal(e)
	}
	pending := catalog.Publication{ID: contracts.ID(), Key: "objects/" + contracts.ID(), Digest: hex.EncodeToString(hash[:]), Size: int64(len(data)), State: "pending"}
	pending, e = s.PublishImmutable(ctx, pending, source, func(next catalog.Publication) (catalog.Publication, error) { return next, stopped })
	if e == nil || pending.UploadID == "" {
		t.Fatal("pending upload missing")
	}
	if e = s.Abort(ctx, pending); e != nil {
		t.Fatal(e)
	}
	if _, e = s.core.ListObjectParts(ctx, bucket, pending.Key, pending.UploadID, 0, 1000); e == nil {
		t.Fatal("aborted upload remains")
	}
}
