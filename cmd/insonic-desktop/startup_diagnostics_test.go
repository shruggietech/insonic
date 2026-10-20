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
	"time"
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

func TestNativeExitTimingIsFixedMonotonicAndDisabledWithoutClockOrOutput(t *testing.T) {
	var out bytes.Buffer
	calls := 0
	clock := func() time.Time { calls++; return time.Unix(0, int64(calls)*int64(time.Millisecond)) }
	disabled := nativeExitStageTimings(false, &out, clock)
	disabled.record("startup-entered")
	if calls != 0 || out.Len() != 0 {
		t.Fatal("disabled timing performed work")
	}
	record := nativeExitStageTimings(true, &out, clock)
	record.record("private-path-or-token")
	record.record("startup-entered")
	record.record("window-prepare-returned")
	if calls != 3 {
		t.Fatal("unknown stage queried clock", calls)
	}
	lines := bytes.Split(bytes.TrimSpace(out.Bytes()), []byte{10})
	if len(lines) != 2 || strings.Contains(out.String(), "private") {
		t.Fatal("unbounded stage retained", out.String())
	}
	for index, line := range lines {
		var value map[string]any
		if json.Unmarshal(line, &value) != nil || value["origin_unix_micros"] != float64(1000) || value["kind"] != "native-exit-stage-timing" || value["elapsed_ns"] != float64(index+1)*float64(time.Millisecond) {
			t.Fatal("monotonic stage differs", string(line))
		}
	}
}

type nativeTimingWriter func([]byte) (int, error)

func (f nativeTimingWriter) Write(value []byte) (int, error) { return f(value) }

func TestNativeExitWatchdogArmsOriginalDeadlineBeforeObservation(t *testing.T) {
	fired := make(chan time.Time, 1)
	armed := false
	var out bytes.Buffer
	writer := nativeTimingWriter(func(value []byte) (int, error) {
		if !armed || len(fired) != 1 {
			t.Fatal("output preceded original timer arming")
		}
		return out.Write(value)
	})
	calls := 0
	clock := func() time.Time { calls++; return time.Unix(0, int64(calls)*int64(time.Millisecond)) }
	record := nativeExitStageTimings(true, writer, clock)
	got := record.watchdog(func(duration time.Duration) <-chan time.Time {
		if duration != 12*time.Second {
			t.Fatal("native deadline changed", duration)
		}
		armed = true
		fired <- time.Unix(0, 0)
		return fired
	})
	if got != fired {
		t.Fatal("original deadline channel replaced")
	}
	var value map[string]any
	if json.Unmarshal(out.Bytes(), &value) != nil || value["arm_begin_ns"] != float64(time.Millisecond) || value["arm_end_ns"] != float64(2*time.Millisecond) {
		t.Fatal("timer creation not bracketed", out.String())
	}
	select {
	case <-got:
	default:
		t.Fatal("ready expiry lost")
	}
	out.Reset()
	calls = 0
	disabled := nativeExitStageTimings(false, &out, clock)
	disabled.watchdog(func(duration time.Duration) <-chan time.Time {
		if duration != 12*time.Second {
			t.Fatal("disabled deadline changed")
		}
		return fired
	})
	if calls != 0 || out.Len() != 0 {
		t.Fatal("disabled watchdog observation performed work")
	}
}
