//go:build integration

// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"testing"
)

func TestGenericS3ArtifactContract(t *testing.T) {
	endpoint := os.Getenv("INSONIC_FIXTURE_S3")
	if endpoint == "" {
		t.Fatal("INSONIC_FIXTURE_S3 required")
	}
	client, e := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4("fixture", "fixture-only", ""), Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
	if e != nil {
		t.Fatal(e)
	}
	bucket := "artifacts-" + contracts.ID()
	if e = client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{Region: "us-east-1"}); e != nil {
		t.Fatal(e)
	}
	defer client.RemoveBucket(context.Background(), bucket)
	s, e := NewS3(context.Background(), S3Config{Endpoint: "http://" + endpoint, Bucket: bucket, Region: "us-east-1", AddressingStyle: "path", Authentication: "anonymous"}, nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	exerciseStore(t, s)
	secondStore, e := NewS3(context.Background(), S3Config{Endpoint: "http://" + endpoint, Bucket: bucket, Region: "us-east-1", AddressingStyle: "path", Authentication: "anonymous"}, nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	sharedStoreIsolation(t, s, secondStore)
	legacyService := testService(t)
	legacyService.Store.Close()
	legacyService.Store = s
	legacyPhysicalRetirementSuite(t, legacyService)
	service := testService(t)
	service.Store.Close()
	service.Store = s
	for _, data := range [][]byte{{}, []byte("S3 seekable\x00input"), bytes.Repeat([]byte("multipart"), 2<<20)} {
		ctx := context.Background()
		p, e := service.Publish(ctx, contracts.ID(), bytesFile(t, data).Name(), "other")
		if e != nil {
			t.Fatal(e)
		}
		m, e := service.Materialize(ctx, p.ID)
		if e != nil {
			t.Fatal(e)
		}
		got, e := os.ReadFile(m.Path)
		if e != nil || !bytes.Equal(got, data) {
			t.Fatal("S3 materialization differs")
		}
		if _, e = service.Retire(ctx, p.ID); e == nil {
			t.Fatal("S3 live lease retired")
		}
		if e = service.Release(ctx, p.ID, m.Lease.ID); e != nil {
			t.Fatal(e)
		}
		if p, e = service.Retire(ctx, p.ID); e != nil || p.State != "retired" {
			t.Fatalf("S3 retirement %+v %v", p, e)
		}
	}
	// A saved destination must route through its own immutable S3 profile while
	// the workspace default remains a filesystem store.
	route := testService(t)
	profile := workspace.Profile{ID: contracts.ID(), Revision: 1, Adapter: "s3", Version: contracts.Version, Configuration: map[string]any{"endpoint": "http://" + endpoint, "bucket": bucket, "region": "us-east-1", "addressing_style": "path", "authentication": "anonymous"}}
	options, _ := json.Marshal(map[string]any{"destination": profile})
	if _, e = route.Catalog.MutateImportPreset(context.Background(), contracts.ID(), "create", 0, catalog.ImportPreset{Kind: "import-preset", Version: contracts.Version, ID: contracts.ID(), Name: "S3 election", Options: options}); e != nil {
		t.Fatal(e)
	}
	p, e := route.PublishTo(context.Background(), profile, contracts.ID(), bytesFile(t, []byte("routed S3 bytes")).Name(), "other")
	if e != nil {
		t.Fatal(e)
	}
	materialized, e := route.Materialize(context.Background(), p.ID)
	if e != nil {
		t.Fatal(e)
	}
	got, e := os.ReadFile(materialized.Path)
	if e != nil || string(got) != "routed S3 bytes" || p.ProfileID != profile.ID {
		t.Fatal("preset S3 route", p, e)
	}
	if e = route.Release(context.Background(), p.ID, materialized.Lease.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = route.Retire(context.Background(), p.ID); e != nil {
		t.Fatal(e)
	}

}
