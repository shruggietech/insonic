// SPDX-License-Identifier: Apache-2.0
//go:build integration

package library

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func backendLibrary(t *testing.T, backend, storage string) *Service {
	t.Helper()
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "backend media fixture")
	if e != nil {
		t.Fatal(e)
	}
	secrets := catalog.SessionSecrets{}
	var postgresConfig catalog.PostgreSQLConfig
	if backend == "postgresql" {
		dsn := os.Getenv("INSONIC_FIXTURE_POSTGRES")
		if dsn == "" {
			t.Fatal("INSONIC_FIXTURE_POSTGRES required")
		}
		cfg, e := pgx.ParseConfig(dsn)
		if e != nil {
			t.Fatal("invalid disposable PostgreSQL fixture")
		}
		schema := "library_" + strings.ReplaceAll(contracts.ID(), "-", "")
		postgresConfig = catalog.PostgreSQLConfig{Host: cfg.Host, Port: cfg.Port, Database: cfg.Database, Schema: schema, TLSMode: "local", CredentialID: "fixture"}
		auth, _ := json.Marshal(catalog.Credentials{Username: cfg.User, Password: cfg.Password})
		secrets["fixture"] = auth
		w.Config.Profiles.Catalog.Adapter = "postgresql"
		raw, _ := json.Marshal(postgresConfig)
		json.Unmarshal(raw, &w.Config.Profiles.Catalog.Configuration)
		t.Cleanup(func() {
			connection, e := pgx.Connect(context.Background(), dsn)
			if e == nil {
				connection.Exec(context.Background(), `DROP SCHEMA "`+schema+`" CASCADE`)
				connection.Close(context.Background())
			}
		})
	}
	if storage == "s3" {
		endpoint := os.Getenv("INSONIC_FIXTURE_S3")
		if endpoint == "" {
			t.Fatal("INSONIC_FIXTURE_S3 required")
		}
		client, e := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4("fixture", "fixture-only", ""), Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
		if e != nil {
			t.Fatal(e)
		}
		bucket := "library-" + contracts.ID()
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
		w.Config.Profiles.Storage.Adapter = "s3"
		w.Config.Profiles.Storage.Configuration = map[string]any{"endpoint": "http://" + endpoint, "bucket": bucket, "region": "us-east-1", "addressing_style": "path", "authentication": "anonymous"}
	}
	var db catalog.Catalog
	if backend == "postgresql" {
		db, e = catalog.OpenPostgreSQL(ctx, postgresConfig, w.Config.WorkspaceID, secrets, true)
	} else {
		db, e = catalog.OpenWorkspace(ctx, w, secrets, false)
	}
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if e = db.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	a, e := artifact.NewService(ctx, w, db, secrets, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { a.Close() })
	tools := Tools{}
	if path := os.Getenv("INSONIC_LIBRARY_TOOLS_FILE"); path != "" {
		data, e := os.ReadFile(path)
		if e != nil || strict(data, &tools) != nil {
			t.Fatal("invalid exact extractor fixture")
		}
	}
	return NewService(a, db, secrets, tools)
}
func TestFullLibraryBackendParity(t *testing.T) {
	for _, backend := range []string{"sqlite", "postgresql"} {
		for _, storage := range []string{"filesystem", "s3"} {
			t.Run(backend+"/"+storage, func(t *testing.T) {
				s := backendLibrary(t, backend, storage)
				ctx := context.Background()
				source := wavFixture(t)
				work := claimLibraryWork(t, s.Catalog, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", OriginatedOn: "2026-10-07"}, Items: []Item{{Source: source}}})
				value, e := s.Execute(ctx, work)
				if e != nil {
					t.Fatal(e)
				}
				result := value.(ImportResult)
				if result.Items[0].State != "admitted" {
					t.Fatalf("admission %+v", result)
				}
				entry, e := s.Catalog.Library(ctx, result.Items[0].MediaID)
				if e != nil {
					t.Fatal(e)
				}
				oldIDs, _ := catalog.PublicationIDs(entry.ReportPublicationIDs)
				work = claimLibraryWork(t, s.Catalog, "media.refresh", RefreshRequest{MediaID: entry.ID})
				current, e := s.Refresh(ctx, work, RefreshRequest{MediaID: entry.ID})
				if e != nil {
					t.Fatal(e)
				}
				if current.AssetID != entry.AssetID || current.Digest != entry.Digest || decodeDates(current.Dates).Selected.Literal != "2026-10-07" {
					t.Fatal("refresh changed original identity or owner date")
				}
				for _, id := range oldIDs {
					p, e := s.Catalog.Publication(ctx, id)
					if e != nil || p.State != "retired" {
						t.Fatal("superseded report state retained")
					}
					if _, e = s.Artifacts.Store.Stat(ctx, p); e == nil {
						t.Fatal("superseded report physically retained")
					}
				}
				if e = s.Artifacts.Verify(ctx, *entry.OriginalPublicationID); e != nil {
					t.Fatal("original retired with report")
				}
				snap, e := s.Catalog.Export(ctx)
				if e != nil || len(snap.Records.Library) != 1 {
					t.Fatal("portable current library missing")
				}
				if len(snap.Records.Library[0].Metadata) == 0 {
					t.Fatal("current metadata missing")
				}
			})
		}
	}
}
