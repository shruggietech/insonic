// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPhysicalIntegrityCategories(t *testing.T) {
	ctx := context.Background()
	sum := sha256.Sum256([]byte("abcdef"))
	p := catalog.Publication{Key: "object", State: "available", Size: 6, Digest: hex.EncodeToString(sum[:])}
	for _, backend := range []string{"filesystem", "s3"} {
		for _, mode := range []string{"valid", "corrupt", "truncated", "missing", "denied"} {
			t.Run(backend+"/"+mode, func(t *testing.T) {
				var store Store
				data := []byte("abcdef")
				if mode == "corrupt" {
					data = []byte("abcdeg")
				}
				if mode == "truncated" {
					data = []byte("abc")
				}
				if backend == "filesystem" {
					if mode == "denied" {
						t.Skip("access denial depends on host ACLs; S3 covers the deterministic denial")
					}
					s, e := NewFilesystem(filepath.Join(t.TempDir(), "objects"))
					if e != nil {
						t.Fatal(e)
					}
					store = s
					if mode != "missing" {
						if e = os.WriteFile(s.root.Name()+"/object", data, 0600); e != nil {
							t.Fatal(e)
						}
					}
				} else {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if mode == "missing" {
							w.WriteHeader(404)
							return
						}
						if mode == "denied" {
							w.WriteHeader(403)
							return
						}
						w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
						w.Header().Set("ETag", `"fixture"`)
						w.Header().Set("Content-Length", strconv.Itoa(len(data)))
						if r.Method != "HEAD" {
							w.Write(data)
						}
					}))
					defer server.Close()
					s, e := NewS3(ctx, S3Config{Endpoint: server.URL, Bucket: "fixture", Region: "us-east-1", AddressingStyle: "path", Authentication: "anonymous"}, nil, nil)
					if e != nil {
						t.Fatal(e)
					}
					store = s
				}
				defer store.Close()
				err := Verify(ctx, store, p)
				if mode == "valid" {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				want := "checksum_mismatch"
				if mode == "denied" {
					want = "unavailable"
				}
				if err == nil || contracts.SafeError(err).Code != want {
					t.Fatalf("want %s, got %v", want, err)
				}
			})
		}
	}
}
