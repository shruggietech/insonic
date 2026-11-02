//go:build integration

// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/contracts"
)

func TestAnswerGenericS3CacheAndExport(t *testing.T) {
	endpoint := os.Getenv("INSONIC_FIXTURE_S3")
	if endpoint == "" {
		t.Skip("explicit S3 fixture required")
	}
	ctx := context.Background()
	client, err := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4("fixture", "fixture-only", ""), Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
	if err != nil {
		t.Fatal(err)
	}
	bucket := "answers-" + contracts.ID()
	if err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		for object := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err == nil {
				_ = client.RemoveObject(ctx, bucket, object.Key, minio.RemoveObjectOptions{})
			}
		}
		_ = client.RemoveBucket(ctx, bucket)
	}()
	store, err := artifact.NewS3(ctx, artifact.S3Config{Endpoint: "http://" + endpoint, Bucket: bucket, Region: "us-east-1", AddressingStyle: "path", Authentication: "anonymous"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := configuredApp(t)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	desktopRecording(t, a, desktopMedia(t, a, false), 0, doc)
	service, err := a.artifactService()
	if err != nil {
		t.Fatal(err)
	}
	service.Store.Close()
	service.Store = store
	r := answers.Request{Version: contracts.Version, Question: "Research source"}
	first, err := a.prepareAnswer(ctx, r)
	if err != nil || first.Package.ID == "" {
		t.Fatal(first, err)
	}
	cached, err := a.prepareAnswer(ctx, r)
	if err != nil || !cached.CacheHit || cached.Package.ID != first.Package.ID {
		t.Fatal("S3 cache", cached, err)
	}
	shown, err := a.inspectAnswer(ctx, first.Package.ID)
	if err != nil || shown.Freshness != "current" || shown.Package.ID != first.Package.ID {
		t.Fatal("S3 export", shown, err)
	}
	answerGeneratedBackendSuite(t, a, r)
}
