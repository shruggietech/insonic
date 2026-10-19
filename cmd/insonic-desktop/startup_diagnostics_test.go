//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
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
