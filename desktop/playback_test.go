// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
)

func playbackFixture(t *testing.T) (*Bridge, *atomic.Bool, string) {
	t.Helper()
	w, err := workspace.Init(t.TempDir(), "Playback fixture")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "native bytes")
	if err = os.WriteFile(path, []byte("0123456789"), 0600); err != nil {
		t.Fatal(err)
	}
	d := playbackDescriptor{PlaybackID: contracts.ID(), MediaID: contracts.ID(), Revision: 1, Path: path, Digest: strings.Repeat("a", 64), SizeBytes: 10, MIMEType: "audio/wav", SourceDigest: strings.Repeat("a", 64), SourceSizeBytes: 10}
	stale := &atomic.Bool{}
	b := &Bridge{Workspace: w}
	b.playbackCall = func(selected *workspace.Workspace, request contracts.Request) contracts.Response {
		wire := request
		wire.Kind, wire.Version, wire.RequestID = "runtime-request", contracts.Version, contracts.ID()
		raw, _ := json.Marshal(wire)
		if err := schemas.ValidateRequest(raw); err != nil {
			t.Fatal("native playback request violates published contract", err)
		}
		if selected != w || request.WorkspaceID != w.Config.WorkspaceID || request.ItemID != d.MediaID {
			t.Error("workspace/media binding lost")
		}
		if stale.Load() && request.Operation == "media.playback-check" {
			return contracts.Response{Error: contracts.Fail("conflict")}
		}
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, Result: d}
	}
	response := b.Playback(d.MediaID, 1, 0, "")
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	raw, _ := json.Marshal(response.Result)
	if strings.Contains(string(raw), path) || strings.Contains(string(raw), "playback_id") || strings.Contains(string(raw), "lease_id") {
		t.Fatal("native playback authority leaked into browser")
	}
	return b, stale, response.Result.(map[string]any)["url"].(string)
}

func TestScopedPlaybackRangesAndCurrentAuthority(t *testing.T) {
	b, stale, url := playbackFixture(t)
	defer ClosePlaybackHandles(b)
	server := PlaybackServer{Bridge: b}
	request := httptest.NewRequest(http.MethodGet, url, nil)
	request.Header.Set("Range", "bytes=2-4")
	result := httptest.NewRecorder()
	server.ServeHTTP(result, request)
	if result.Code != http.StatusPartialContent || result.Body.String() != "234" || result.Header().Get("Content-Range") != "bytes 2-4/10" || result.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("range contract", result.Code, result.Header(), result.Body.String())
	}
	head := httptest.NewRecorder()
	server.ServeHTTP(head, httptest.NewRequest(http.MethodHead, url, nil))
	if head.Code != http.StatusOK || head.Body.Len() != 0 || head.Header().Get("Content-Length") != "10" {
		t.Fatal("HEAD contract")
	}
	stale.Store(true)
	if current := b.VerifyPlayback(url); current.Error == nil {
		t.Fatal("superseded buffered playback remained current")
	}
	replaced := httptest.NewRecorder()
	server.ServeHTTP(replaced, httptest.NewRequest(http.MethodGet, url, nil))
	if replaced.Code != http.StatusGone || replaced.Body.Len() != 0 {
		t.Fatal("superseded media served")
	}
}

func TestScopedPlaybackRejectsArbitraryPathsMethodsAndSwitches(t *testing.T) {
	b, _, url := playbackFixture(t)
	defer ClosePlaybackHandles(b)
	server := PlaybackServer{Bridge: b}
	for _, bad := range []string{"/media/../../private", "/media/" + contracts.ID(), url + "?path=private", "/etc/passwd"} {
		result := httptest.NewRecorder()
		server.ServeHTTP(result, httptest.NewRequest(http.MethodGet, bad, nil))
		if result.Code == http.StatusOK || result.Code == http.StatusPartialContent {
			t.Fatal("unscoped path served", bad)
		}
	}
	result := httptest.NewRecorder()
	server.ServeHTTP(result, httptest.NewRequest(http.MethodPost, url, nil))
	if result.Code != http.StatusMethodNotAllowed {
		t.Fatal("mutation method admitted")
	}
	if b.SelectWorkspace(filepath.Join(t.TempDir(), "new"), "Next", true).Error != nil {
		t.Fatal("switch")
	}
	result = httptest.NewRecorder()
	server.ServeHTTP(result, httptest.NewRequest(http.MethodGet, url, nil))
	if result.Code != http.StatusGone {
		t.Fatal("old workspace ticket survived")
	}
}

func TestScopedPlaybackRechecksSelectionAfterRuntimeReadback(t *testing.T) {
	b, _, url := playbackFixture(t)
	defer ClosePlaybackHandles(b)
	original := b.playbackCall
	next, err := workspace.Init(filepath.Join(t.TempDir(), "next"), "Next")
	if err != nil {
		t.Fatal(err)
	}
	b.playbackCall = func(selected *workspace.Workspace, request contracts.Request) contracts.Response {
		response := original(selected, request)
		if request.Operation == "media.playback-check" {
			b.mu.Lock()
			b.Workspace = next
			b.mu.Unlock()
		}
		return response
	}
	result := httptest.NewRecorder()
	PlaybackServer{Bridge: b}.ServeHTTP(result, httptest.NewRequest(http.MethodGet, url, nil))
	if result.Code != http.StatusGone || result.Body.Len() != 0 {
		t.Fatal("runtime readback from previously selected workspace served bytes")
	}
}
