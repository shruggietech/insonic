//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type startupAssetCounts struct {
	started  [3]atomic.Uint64
	returned [3]atomic.Uint64
}

var qualificationStartupAssets atomic.Pointer[startupAssetCounts]

// Observe only disposable qualification. Preserve the original writer, request,
// policy, handler ordering and errors. No URL or response content is retained.
func observeQualificationAssets(policy func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	progress := &startupAssetCounts{}
	qualificationStartupAssets.Store(progress)
	return func(next http.Handler) http.Handler {
		handler := policy(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			kind := 2
			if r.URL.Path == "/" || r.URL.Path == "/index.html" {
				kind = 0
			} else if strings.HasSuffix(r.URL.Path, ".js") {
				kind = 1
			}
			progress.started[kind].Add(1)
			defer progress.returned[kind].Add(1)
			handler.ServeHTTP(w, r)
		})
	}
}
func startupAssetSnapshot() map[string]uint64 {
	progress := qualificationStartupAssets.Load()
	if progress == nil {
		return nil
	}
	result := map[string]uint64{}
	for i, name := range []string{"document", "script", "other"} {
		result[name+"_entered"] = progress.started[i].Load()
		result[name+"_returned"] = progress.returned[i].Load()
	}
	return result
}
func logNativeStartupProgress() {
	writeNativeStartupProgress(os.Stderr, nativeStartupWaitSnapshot)
}

func writeNativeStartupProgress(out io.Writer, observeWaits func() any) {
	snapshot := map[string]any{"kind": "native-startup-progress", "unix_millis": time.Now().UnixMilli(), "assets": startupAssetSnapshot(), "windows": nativeWebviewBootstrapSnapshot()}
	raw, _ := json.Marshal(snapshot)
	fmt.Fprintln(out, string(raw))
	// Optional native diagnostics cannot delay or erase the primary counters.
	waits := observeWaits()
	raw, _ = json.Marshal(map[string]any{"kind": "native-startup-waits", "unix_millis": time.Now().UnixMilli(), "waits": waits})
	fmt.Fprintln(out, string(raw))
}
