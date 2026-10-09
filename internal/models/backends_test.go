// SPDX-License-Identifier: Apache-2.0
//go:build integration

package models

import (
	"context"
	"encoding/json"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestS3ModelAcquisitionMissingByteRecovery(t *testing.T) {
	ctx := context.Background()
	endpoint := os.Getenv("INSONIC_FIXTURE_S3")
	if endpoint == "" {
		t.Fatal("INSONIC_FIXTURE_S3 required")
	}
	client, e := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4("fixture", "fixture-only", ""), Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
	if e != nil {
		t.Fatal(e)
	}
	bucket := "models-" + contracts.ID()
	if e = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"}); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		for object := range client.ListObjects(context.Background(), bucket, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err == nil {
				client.RemoveObject(context.Background(), bucket, object.Key, minio.RemoveObjectOptions{})
			}
		}
		client.RemoveBucket(context.Background(), bucket)
	})
	w, e := workspace.Init(t.TempDir(), "S3 model fixture")
	if e != nil {
		t.Fatal(e)
	}
	w.Config.Profiles.Storage.Adapter = "s3"
	w.Config.Profiles.Storage.Configuration = map[string]any{"endpoint": "http://" + endpoint, "bucket": bucket, "region": "us-east-1", "addressing_style": "path", "authentication": "anonymous"}
	db, e := catalog.OpenWorkspace(ctx, w, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = db.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	a, e := artifact.NewService(ctx, w, db, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	service := NewService(a, db, nil)
	body := []byte("S3 synthetic model bundle")
	server := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, r *http.Request) { out.Write(body) }))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/weights", body)}
	claim := modelWork(t, db, "models.acquire", m)
	id := completedModel(t, service, claim)
	current, e := db.BaseModel(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	ids, _ := catalog.PublicationIDs(current.PublicationIDs)
	p, e := db.Publication(ctx, ids[0])
	if e != nil {
		t.Fatal(e)
	}
	if e = client.RemoveObject(ctx, bucket, p.Key, minio.RemoveObjectOptions{}); e != nil {
		t.Fatal(e)
	}
	if service.Verify(ctx, id) == nil {
		t.Fatal("missing S3 bytes accepted")
	}
	if _, e = db.CheckpointWork(ctx, claim, "complete", "succeeded", json.RawMessage(`{"state":"available"}`), time.Minute); e != nil {
		t.Fatal(e)
	}
	retry, e := db.RetryWork(ctx, contracts.ID(), claim.ID)
	if e != nil {
		t.Fatal(e)
	}
	retry, e = db.ClaimWork(ctx, retry.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = service.Execute(ctx, retry); e != nil {
		t.Fatal(e)
	}
	if e = service.Verify(ctx, id); e != nil {
		t.Fatal(e)
	}
	leases, e := service.Materialize(ctx, id)
	if e != nil || len(leases) != 1 {
		t.Fatal("S3 materialization", e)
	}
	for _, lease := range leases {
		if e = a.Release(ctx, lease.PublicationID, lease.Lease.ID); e != nil {
			t.Fatal(e)
		}
	}
}
