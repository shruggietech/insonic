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
	snapshot := map[string]any{"kind": "native-startup-progress", "unix_millis": time.Now().UnixMilli(), "assets": startupAssetSnapshot(), "windows": nativeWebviewBootstrapSnapshot(), "browser": nativeWebviewBrowserSnapshot()}
	raw, _ := json.Marshal(snapshot)
	fmt.Fprintln(out, string(raw))
	// Optional native diagnostics cannot delay or erase the primary counters.
	waits := observeWaits()
	raw, _ = json.Marshal(map[string]any{"kind": "native-startup-waits", "unix_millis": time.Now().UnixMilli(), "waits": waits})
	fmt.Fprintln(out, string(raw))
}

// Elected browser diagnosis retains a passing reference without native queries.
func logNativeBrowserReadyProgress() {
	if os.Getenv("INSONIC_STARTUP_BROWSER") != "true" {
		return
	}
	raw, _ := json.Marshal(map[string]any{"kind": "native-browser-ready-progress", "unix_millis": time.Now().UnixMilli(), "browser": nativeWebviewBrowserSnapshot()})
	fmt.Fprintln(os.Stderr, string(raw))
}

// Fixed qualification stages use the child's own monotonic clock. This observer
// adds no browser calls and is elected only by the disposable live diagnosis.
type nativeExitTimings struct {
	origin time.Time
	out    io.Writer
	clock  func() time.Time
}

func nativeExitStageTimings(enabled bool, out io.Writer, clock func() time.Time) *nativeExitTimings {
	if !enabled {
		return nil
	}
	return &nativeExitTimings{origin: clock(), out: out, clock: clock}
}

func (e *nativeExitTimings) record(stage string) {
	if e == nil {
		return
	}
	switch stage {
	case "controller-startup", "startup-entered", "window-prepare-returned", "watchdog-timeout", "dom-ready", "shutdown":
	default:
		return
	}
	elapsed := e.clock().Sub(e.origin)
	if elapsed < 0 {
		return
	}
	e.emit(map[string]any{"stage": stage, "elapsed_ns": elapsed.Nanoseconds()})
}

func (e *nativeExitTimings) emit(value map[string]any) {
	value["kind"] = "native-exit-stage-timing"
	value["origin_unix_micros"] = e.origin.UnixMicro()
	raw, _ := json.Marshal(value)
	fmt.Fprintln(e.out, string(raw))
}

func (e *nativeExitTimings) watchdog(after func(time.Duration) <-chan time.Time) <-chan time.Time {
	if e == nil {
		return after(12 * time.Second)
	}
	// Bracket timer creation without output between these clock reads. A delayed
	// observer cannot silently turn its later receipt time into the arm time.
	before := e.clock().Sub(e.origin)
	deadline := after(12 * time.Second)
	finished := e.clock().Sub(e.origin)
	if before >= 0 && finished >= before {
		e.emit(map[string]any{"stage": "watchdog-armed", "elapsed_ns": finished.Nanoseconds(),
			"arm_begin_ns": before.Nanoseconds(), "arm_end_ns": finished.Nanoseconds()})
	}
	return deadline
}
