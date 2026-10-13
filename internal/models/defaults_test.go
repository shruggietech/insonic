// SPDX-License-Identifier: Apache-2.0
package models

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestDefaultModelRolesAndIdentities(t *testing.T) {
	for _, c := range Defaults() {
		if c.Manifest.Validate() != nil || c.Bytes() < 1 {
			t.Fatal("invalid builtin")
		}
		op := "transcription"
		if c.Role == "diarization" {
			op = "diarization"
		}
		if CheckCompatibility(c.Manifest, op, "", "") != nil {
			t.Fatal("incompatible builtin")
		}
		if len(c.Manifest.Revision) != 40 {
			t.Fatal("mutable default")
		}
	}
}
func TestPinnedHFRedirectNeverCarriesAuthorizationOffOrigin(t *testing.T) {
	u, _ := url.Parse("https://huggingface.co/org/model/resolve/0123456789012345678901234567890123456789/model.bin")
	first := &http.Request{URL: u, Header: http.Header{"Authorization": []string{"Bearer private"}}}
	nextURL, _ := url.Parse("https://cas-bridge.xethub.hf.co/content?X-Amz-Signature=transient")
	next := &http.Request{URL: nextURL, Header: first.Header.Clone()}
	if boundedRedirect(next, []*http.Request{first}) != nil || next.Header.Get("Authorization") != "" {
		t.Fatal("credential leaked or valid CDN rejected")
	}
	for _, host := range []string{"us.aws.cdn.hf.co", "us.gcp.cdn.hf.co"} {
		cdn, _ := url.Parse("https://" + host + "/content?Signature=transient")
		request := &http.Request{URL: cdn, Header: first.Header.Clone()}
		if boundedRedirect(request, []*http.Request{first}) != nil || request.Header.Get("Authorization") != "" {
			t.Fatal("official CDN failed", host)
		}
	}
	for _, raw := range []string{"https://evil.example/model", "http://cas-bridge.xethub.hf.co/model", "https://huggingface.co.evil.example/model"} {
		v, _ := url.Parse(raw)
		if boundedRedirect(&http.Request{URL: v, Header: first.Header.Clone()}, []*http.Request{first}) == nil {
			t.Fatal("unapproved redirect", raw)
		}
	}
	custom, _ := url.Parse("https://example.org/model")
	if boundedRedirect(next, []*http.Request{{URL: custom}}) == nil {
		t.Fatal("custom source received HF privileges")
	}
}
func TestLocalModelReusePublishesVerifiedBytesWithoutNetwork(t *testing.T) {
	s, db := modelServiceFixture(t)
	dir := t.TempDir()
	m := fixtureManifest()
	os.WriteFile(filepath.Join(dir, "weights"), []byte("model fixture"), 0600)
	req := Request{Manifest: m, LocalDirectory: dir, Destination: &s.Artifacts.Workspace.Config.Profiles.Storage}
	raw, _ := json.Marshal(req)
	w, e := db.EnqueueWork(context.Background(), contracts.ID(), "models.acquire", raw)
	if e != nil {
		t.Fatal(e)
	}
	w, e = db.ClaimWork(context.Background(), w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Execute(context.Background(), w); e != nil {
		t.Fatal(e)
	}
	if s.Verify(context.Background(), InstallationID(m)) != nil {
		t.Fatal("local reuse not verified")
	}
	b, e := os.ReadFile(filepath.Join(dir, "weights"))
	if e != nil || string(b) != "model fixture" {
		t.Fatal("source changed")
	}
}

func TestAcquisitionKeepsFrozenDestinationAfterWorkspaceChange(t *testing.T) {
	s, db := modelServiceFixture(t)
	dir := t.TempDir()
	m := fixtureManifest()
	os.WriteFile(filepath.Join(dir, "weights"), []byte("model fixture"), 0600)
	frozen := s.Artifacts.Workspace.Config.Profiles.Storage
	req := Request{Manifest: m, LocalDirectory: dir, Destination: &frozen}
	raw, _ := json.Marshal(req)
	w, e := db.EnqueueWork(context.Background(), contracts.ID(), "models.acquire", raw)
	if e != nil {
		t.Fatal(e)
	}
	w, e = db.ClaimWork(context.Background(), w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	s.Artifacts.Workspace.Config.Profiles.Storage.ID = contracts.ID()
	s.Artifacts.Workspace.Config.Profiles.Storage.Configuration = map[string]any{"root": t.TempDir()}
	if _, e = s.Execute(context.Background(), w); e != nil {
		t.Fatal(e)
	}
	install, e := db.BaseModel(context.Background(), InstallationID(m))
	if e != nil {
		t.Fatal(e)
	}
	var ids []string
	json.Unmarshal(install.PublicationIDs, &ids)
	for _, id := range ids {
		p, e := db.Publication(context.Background(), id)
		if e != nil || p.ProfileID != frozen.ID || p.ProfileRevision != frozen.Revision {
			t.Fatal("destination drifted", p, e)
		}
	}
}

func TestAssetDownloadReportsBoundedBytesAndCachedCompletion(t *testing.T) {
	s, _ := modelServiceFixture(t)
	body := []byte("verified asset bytes")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; w.Write(body) }))
	defer server.Close()
	file := File{Role: "asset", URL: server.URL, LocalHTTP: true, Size: int64(len(body)), SHA256: fmt.Sprintf("%x", sha256.Sum256(body))}
	id := contracts.ID()
	for attempt := 0; attempt < 2; attempt++ {
		progress := []int64{}
		_, e := s.DownloadAsset(context.Background(), id, file, func(n int64) error { progress = append(progress, n); return nil })
		if e != nil || len(progress) == 0 || progress[len(progress)-1] != file.Size {
			t.Fatal(progress, e)
		}
		for _, n := range progress {
			if n < 0 || n > file.Size {
				t.Fatal("invented byte progress", n)
			}
		}
	}
	if requests != 1 {
		t.Fatal("verified cache redownloaded", requests)
	}
}

func TestLocalLinkedSnapshotReusesManifestBytes(t *testing.T) {
	s, db := modelServiceFixture(t)
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "blob")
	os.WriteFile(target, []byte("model fixture"), 0600)
	if e := os.Symlink(target, filepath.Join(dir, "weights")); e != nil {
		t.Skip("host does not allow symlink creation")
	}
	model := fixtureManifest()
	if e := VerifyLocal(context.Background(), dir, model); e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(Request{Manifest: model, LocalDirectory: dir})
	w, e := db.EnqueueWork(context.Background(), contracts.ID(), "models.acquire", raw)
	if e == nil {
		w, e = db.ClaimWork(context.Background(), w.ID, contracts.ID(), time.Minute)
	}
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Execute(context.Background(), w); e != nil {
		t.Fatal(e)
	}
	if e = s.Verify(context.Background(), InstallationID(model)); e != nil {
		t.Fatal(e)
	}
}
