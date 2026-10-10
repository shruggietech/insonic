//go:build integration

// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestPortableRestoreResumesExactS3MultipartUpload(t *testing.T) {
	ctx := context.Background()
	w, source, service := fixture(t)
	p := publish(t, service, strings.Repeat("multipart-byte", 750000), "canonical-audio")
	dir := filepath.Join(t.TempDir(), "multipart-bundle")
	m, e := Create(ctx, w, source, nil, CreateOptions{ID: contracts.ID(), Directory: dir})
	if e != nil {
		t.Fatal(e)
	}
	target, provider := backendWorkspace(t, "sqlite", "s3")
	upstream, e := url.Parse(target.Config.Profiles.Storage.Configuration["endpoint"].(string))
	if e != nil {
		t.Fatal(e)
	}
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	first, cancel := context.WithCancel(ctx)
	defer cancel()
	var interrupted atomic.Bool
	var starts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == "POST" && request.URL.Query().Has("uploads") {
			starts.Add(1)
		}
		if request.Method == "PUT" && request.URL.Query().Get("partNumber") == "2" && interrupted.CompareAndSwap(false, true) {
			cancel()
			http.Error(response, "injected interruption", http.StatusServiceUnavailable)
			return
		}
		proxy.ServeHTTP(response, request)
	}))
	defer server.Close()
	target.Config.Profiles.Storage.Configuration["endpoint"] = server.URL
	if _, e = RestoreWorkspace(first, target, provider, dir); e == nil {
		t.Fatal("interrupted multipart restore activated")
	}
	if !interrupted.Load() {
		t.Fatal("multipart path not reached")
	}
	progress, e := os.ReadFile(filepath.Join(target.Control, "backup-restore-progress", m.ID, p.ID+".json"))
	if e != nil {
		t.Fatal(e)
	}
	var journal restoreProgress
	if strict(progress, &journal) != nil || journal.Publication.UploadID == "" || len(journal.Publication.Parts) != 1 {
		t.Fatal("first completed part not durable")
	}
	if _, e = RestoreWorkspace(ctx, target, provider, dir); e != nil {
		t.Fatal("multipart retry", e)
	}
	if starts.Load() != 1 {
		t.Fatal("retry opened a competing multipart upload", starts.Load())
	}
	db, e := catalog.OpenWorkspace(ctx, target, provider, false)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	artifacts, e := artifact.NewService(ctx, target, db, provider, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	defer artifacts.Close()
	if e = artifacts.Verify(ctx, p.ID); e != nil {
		t.Fatal("retrieved multipart bytes", e)
	}
	if _, e = os.Stat(filepath.Join(target.Control, "backup-restore-progress", m.ID)); !os.IsNotExist(e) {
		t.Fatal("completed progress not removed")
	}
}

func backendWorkspace(t *testing.T, backend, storage string) (*workspace.Workspace, catalog.SessionSecrets) {
	t.Helper()
	ctx := context.Background()
	w := destination(t)
	provider := catalog.SessionSecrets{}
	if backend == "postgresql" {
		dsn := os.Getenv("INSONIC_FIXTURE_POSTGRES")
		if dsn == "" {
			t.Fatal("PostgreSQL fixture required")
		}
		c, e := pgx.ParseConfig(dsn)
		if e != nil {
			t.Fatal(e)
		}
		schema := "backup_" + strings.ReplaceAll(contracts.ID(), "-", "")
		id := contracts.ID()
		raw, _ := json.Marshal(catalog.Credentials{Username: c.User, Password: c.Password})
		provider[id] = raw
		config := catalog.PostgreSQLConfig{Host: c.Host, Port: c.Port, Database: c.Database, Schema: schema, TLSMode: "local", CredentialID: id}
		raw, _ = json.Marshal(config)
		w.Config.Profiles.Catalog.Adapter = backend
		// Unmarshal into a fresh map: decoding into the existing SQLite profile
		// retains its path key, which strict PostgreSQL configuration rejects.
		var configuration map[string]any
		if e = json.Unmarshal(raw, &configuration); e != nil {
			t.Fatal(e)
		}
		w.Config.Profiles.Catalog.Configuration = configuration
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
			t.Fatal("S3 fixture required")
		}
		client, e := minio.New(endpoint, &minio.Options{Creds: credentials.NewStaticV4("fixture", "fixture-only", ""), Region: "us-east-1", BucketLookup: minio.BucketLookupPath})
		if e != nil {
			t.Fatal(e)
		}
		bucket := "backup-" + contracts.ID()
		if e = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: "us-east-1"}); e != nil {
			t.Fatal(e)
		}
		w.Config.Profiles.Storage.Adapter = "s3"
		w.Config.Profiles.Storage.Configuration = map[string]any{"endpoint": "http://" + endpoint, "bucket": bucket, "region": "us-east-1", "addressing_style": "path", "authentication": "anonymous"}
		t.Cleanup(func() {
			for object := range client.ListObjects(context.Background(), bucket, minio.ListObjectsOptions{Recursive: true}) {
				if object.Err == nil {
					client.RemoveObject(context.Background(), bucket, object.Key, minio.RemoveObjectOptions{})
				}
			}
			client.RemoveBucket(context.Background(), bucket)
		})
	}
	return w, provider
}

func TestPortableBackupBackendMatrix(t *testing.T) {
	for _, sourceBackend := range []string{"sqlite", "postgresql"} {
		for _, sourceStorage := range []string{"filesystem", "s3"} {
			t.Run(sourceBackend+"/"+sourceStorage, func(t *testing.T) {
				ctx := context.Background()
				w, provider := backendWorkspace(t, sourceBackend, sourceStorage)
				source, e := catalog.OpenWorkspace(ctx, w, provider, true)
				if e != nil {
					t.Fatal(e)
				}
				defer source.Close()
				if e = source.RegisterWorkspace(ctx, w); e != nil {
					t.Fatal(e)
				}
				service, e := artifact.NewService(ctx, w, source, provider, contracts.ID())
				if e != nil {
					t.Fatal(e)
				}
				defer service.Close()
				publication := publish(t, service, "canonical exact stereo sourceclock", "canonical-audio")
				model := publish(t, service, "independent durable weights", "model")
				sp, e := source.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Backup speaker"}})
				if e != nil {
					t.Fatal(e)
				}
				for _, mode := range []string{"self-contained", "reference-only"} {
					dir := filepath.Join(t.TempDir(), "portable")
					m, e := Create(ctx, w, source, provider, CreateOptions{ID: contracts.ID(), Directory: dir, Mode: mode})
					if e != nil {
						t.Fatal("create", e)
					}
					if _, e = Verify(ctx, dir, provider); e != nil {
						t.Fatal("verify", e)
					}
					for _, destinationBackend := range []string{"sqlite", "postgresql"} {
						for _, destinationStorage := range []string{"filesystem", "s3"} {
							t.Run(mode+"/"+destinationBackend+"/"+destinationStorage, func(t *testing.T) {
								target, targetProvider := backendWorkspace(t, destinationBackend, destinationStorage)
								for id, value := range provider {
									targetProvider[id] = value
								}
								if _, e := RestoreWorkspace(ctx, target, targetProvider, dir); e != nil {
									t.Fatal("restore", e)
								}
								db, e := catalog.OpenWorkspace(ctx, target, targetProvider, false)
								if e != nil {
									t.Fatal(e)
								}
								defer db.Close()
								if e = db.RegisterWorkspace(ctx, target); e != nil {
									t.Fatal("restart profile parity", e)
								}
								a, e := artifact.NewService(ctx, target, db, targetProvider, contracts.ID())
								if e != nil {
									t.Fatal(e)
								}
								defer a.Close()
								for _, p := range []catalog.Publication{publication, model} {
									if e = a.Verify(ctx, p.ID); e != nil {
										t.Fatal("bytes", e)
									}
									got, e := db.Publication(ctx, p.ID)
									if e != nil || got.ProfileID != target.Config.Profiles.Storage.ID || got.References[m.ID] {
										t.Fatal("destination authority", e)
									}
								}
								if got, e := db.Speaker(ctx, sp.Speaker.ID); e != nil || got.Speaker.Name != sp.Speaker.Name {
									t.Fatal("speaker identity", e)
								}
								snapshot, e := db.Export(ctx)
								if e != nil {
									t.Fatal(e)
								}
								if e = catalog.ValidateSnapshot(ctx, snapshot); e != nil {
									t.Fatal("portable restored snapshot", e)
								}
							})
						}
					}
					if e = Release(ctx, source, dir); e != nil {
						t.Fatal("release", e)
					}
				}
			})
		}
	}
}
