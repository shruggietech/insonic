// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/secrets"
)

func TestProvisionRequiresS3BucketAccess(t *testing.T) {
	for _, phase := range []string{"present", "absent", "denied", "unreachable", "cancelled"} {
		t.Run(phase, func(t *testing.T) {
			hits := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				if r.Method != http.MethodHead || strings.TrimRight(r.URL.Path, "/") != "/fixture" {
					t.Errorf("unexpected bucket operation: %s %s", r.Method, r.URL.Path)
				}
				if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
					t.Error("protected credential not used")
				}
				switch phase {
				case "absent":
					w.WriteHeader(404)
				case "denied":
					w.Header().Set("X-Amz-Error-Code", "AccessDenied")
					w.WriteHeader(403)
				case "cancelled":
					<-r.Context().Done()
				default:
					w.WriteHeader(200)
				}
			}))
			defer server.Close()
			if phase == "unreachable" {
				server.Close()
			}
			id := contracts.ID()
			raw, _ := json.Marshal(map[string]any{"display_name": "Bucket fixture", "profiles": map[string]any{"storage": map[string]any{"adapter_id": "s3", "configuration": map[string]any{"endpoint": server.URL, "bucket": "fixture", "region": "test", "addressing_style": "path", "authentication": "credential", "credential_id": id}}}})
			c, e := Build(raw)
			if e != nil {
				t.Fatal(e)
			}
			input := &credentialcmd.Input{Session: map[string][]byte{id: []byte("{\"access_key\":\"fixture-key\",\"secret_key\":\"private-bucket-secret\"}")}}
			defer input.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if phase == "cancelled" {
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
			}
			root := t.TempDir()
			_, e = (Provisioner{GraphFactory: fixtureFactory}).ProvisionMode(ctx, root, c, input, "session")
			if phase == "present" {
				if e != nil || hits != 1 {
					t.Fatal(e, hits)
				}
				return
			}
			if e == nil {
				t.Fatal("unusable bucket reported provisioned")
			}
			if strings.Contains(e.Error(), "private-bucket-secret") {
				t.Fatal("credential in error")
			}
			if phase != "unreachable" && hits == 0 {
				t.Fatal("no bucket check")
			}
			if _, e = os.Stat(filepath.Join(root, ".insonic", "workspace.json")); !os.IsNotExist(e) {
				t.Fatal("bucket failure published workspace")
			}
		})
	}
}

func TestCompletedProvisionRetryRequiresCredentialBackendIdentity(t *testing.T) {
	for _, mode := range []string{"native", "vault", "session"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := Build([]byte("{\"display_name\":\"Retry fixture\"}"))
			root := t.TempDir()
			svc := Provisioner{GraphFactory: fixtureFactory}
			w, e := svc.ProvisionMode(context.Background(), root, c, nil, mode)
			if e != nil {
				t.Fatal(e)
			}
			original, _ := os.ReadFile(filepath.Join(w.Control, "workspace.json"))
			intent, _ := os.ReadFile(filepath.Join(w.Control, "provision-intent.json"))
			for _, retryMode := range []string{"native", "vault", "session"} {
				_, e = svc.ProvisionMode(context.Background(), root, c, nil, retryMode)
				if retryMode == mode {
					if e != nil {
						t.Fatal("identical completed retry", e)
					}
				} else if e == nil || contracts.SafeError(e).Code != "conflict" {
					t.Fatal("different completed credential mode accepted", retryMode, e)
				}
			}
			selection, e := secrets.ReadSelection(w)
			if e != nil || selection != mode {
				t.Fatal(selection, e)
			}
			now, _ := os.ReadFile(filepath.Join(w.Control, "workspace.json"))
			retained, _ := os.ReadFile(filepath.Join(w.Control, "provision-intent.json"))
			if string(now) != string(original) || string(retained) != string(intent) {
				t.Fatal("completed retry changed receipt or config")
			}
			changed := "session"
			if mode == changed {
				changed = "vault"
			}
			if e = secrets.Select(w, changed); e != nil {
				t.Fatal(e)
			}
			if _, e = svc.ProvisionMode(context.Background(), root, c, nil, mode); e == nil || contracts.SafeError(e).Code != "conflict" {
				t.Fatal("changed persisted selection reconciled", e)
			}
		})
	}
}

func TestCompletedProvisionRequiresRetainedReceipt(t *testing.T) {
	for _, phase := range []string{"missing", "changed"} {
		t.Run(phase, func(t *testing.T) {
			c, _ := Build([]byte("{\"display_name\":\"Receipt fixture\"}"))
			root := t.TempDir()
			svc := Provisioner{GraphFactory: fixtureFactory}
			w, e := svc.ProvisionMode(context.Background(), root, c, nil, "native")
			if e != nil {
				t.Fatal(e)
			}
			file := filepath.Join(w.Control, "provision-intent.json")
			if phase == "missing" {
				e = os.Remove(file)
			} else {
				e = os.WriteFile(file, []byte("{}"), 0600)
			}
			if e != nil {
				t.Fatal(e)
			}
			if _, e = svc.ProvisionMode(context.Background(), root, c, nil, "native"); e == nil || contracts.SafeError(e).Code != "conflict" {
				t.Fatal("unproven completed receipt accepted", e)
			}
		})
	}
}
