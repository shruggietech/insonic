//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestStartupAssetProgressTracksPendingHandlerWithoutRetainingRequests(t *testing.T) {
	prior := qualificationStartupAssets.Load()
	defer qualificationStartupAssets.Store(prior)
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var once sync.Once
	releaseHandler := func() { once.Do(func() { close(release) }) }
	defer func() { releaseHandler(); <-done }()
	writer := httptest.NewRecorder()
	middleware := observeQualificationAssets(func(next http.Handler) http.Handler { return next })
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if w != writer {
			t.Error("startup observation replaced response writer")
		}
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	go func() {
		defer close(done)
		handler.ServeHTTP(writer, httptest.NewRequest("GET", "http://controlled.invalid/private-asset.js?token=private-value", nil))
	}()
	<-entered
	during := startupAssetSnapshot()
	if during["script_entered"] != 1 || during["script_returned"] != 0 {
		t.Fatal("pending asset work not distinguished", during)
	}
	raw, _ := json.Marshal(during)
	if strings.Contains(string(raw), "private") || strings.Contains(string(raw), "token") || strings.Contains(string(raw), "controlled") {
		t.Fatal("startup observation retained request data")
	}
	releaseHandler()
	<-done
	if startupAssetSnapshot()["script_returned"] != 1 || writer.Code != http.StatusNoContent {
		t.Fatal("handler return or status changed")
	}
}

func TestStartupPrimaryProgressPrecedesOptionalNativeObservation(t *testing.T) {
	var out bytes.Buffer
	writeNativeStartupProgress(&out, func() any {
		var primary map[string]any
		if json.Unmarshal(bytes.TrimSpace(out.Bytes()), &primary) != nil || primary["kind"] != "native-startup-progress" {
			t.Error("optional native work began before the primary counters were written")
		}
		return []any{map[string]any{"role": "host", "thread": 42, "complete": false, "reason": "observer-failed"}}
	})
	lines := bytes.Split(bytes.TrimSpace(out.Bytes()), []byte{'\n'})
	if len(lines) != 2 {
		t.Fatalf("primary and optional observation were not separate records: %s", out.Bytes())
	}
	var primary, optional map[string]any
	if json.Unmarshal(lines[0], &primary) != nil || primary["kind"] != "native-startup-progress" {
		t.Fatal("primary record lost", out.String())
	}
	if _, exists := primary["waits"]; exists {
		t.Fatal("primary record depends on optional native work", out.String())
	}
	if json.Unmarshal(lines[1], &optional) != nil || optional["kind"] != "native-startup-waits" {
		t.Fatal("optional record lost", out.String())
	}
}
