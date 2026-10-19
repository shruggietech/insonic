//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/shruggietech/insonic/internal/process"
	"golang.org/x/sys/windows"
)

func TestNativeWaitMetadataABIAndObjectNameRedaction(t *testing.T) {
	var node nativeWaitNode
	if unsafe.Sizeof(node) != 280 || unsafe.Offsetof(node.Data) != 8 || unsafe.Alignof(node) != 8 {
		t.Fatal("native wait-node ABI differs")
	}
	node.Kind = 3
	node.Status = 3
	data := unsafe.Slice((*uint16)(unsafe.Pointer(&node.Data[0])), 136)
	for i, c := range []uint16{'p', 'r', 'i', 'v', 'a', 't', 'e', '-', 'n', 'a', 'm', 'e'} {
		data[i] = c
	}
	thread := nativeWaitNode{Kind: 8, Status: 2}
	thread.Data[0] = uint64(123) | uint64(456)<<32
	rows := nativeWaitItems([]nativeWaitNode{node, thread})
	raw, _ := json.Marshal(rows)
	if strings.Contains(string(raw), "private") || strings.Contains(string(raw), "name") || rows[0].PID != 0 || rows[1].PID != 123 || rows[1].Thread != 456 {
		t.Fatal("wait metadata leaked object data or lost thread identity", string(raw))
	}
	if len(nativeWaitItems(make([]nativeWaitNode, 20))) != 16 {
		t.Fatal("wait metadata exceeded bound")
	}
}

// The test executable exercises the real hidden process boundary without Wails,
// windows, input, or an in-process native query in the test host.
func TestNativeWaitObserverHelperProcess(t *testing.T) {
	mode := os.Getenv("INSONIC_NATIVE_WAIT_HELPER_MODE")
	if mode == "" {
		return
	}
	switch mode {
	case "crash":
		_ = windows.TerminateProcess(windows.CurrentProcess(), 0xc0000005)
		os.Exit(99)
	case "timeout":
		time.Sleep(time.Minute)
		os.Exit(99)
	case "malformed":
		fmt.Fprintln(os.Stdout, "private-native-dump")
		fmt.Fprintln(os.Stderr, "private-native-stack")
		os.Exit(0)
	case "truncated":
		fmt.Fprint(os.Stdout, strings.Repeat("private-native-dump", 1000))
		os.Exit(0)
	case "wrong-target", "too-many-nodes", "private-field":
		var targets []nativeWaitTarget
		if !decodeNativeWaitJSON(os.Stdin, 1024, &targets) || !validNativeWaitTargets(targets) {
			os.Exit(98)
		}
		value := nativeWaitObservation{Role: targets[0].Role, Thread: targets[0].Thread, Reason: "api-unavailable"}
		if mode == "wrong-target" {
			value.Thread++
		}
		if mode == "too-many-nodes" {
			value.Nodes = make([]nativeWaitItem, 17)
		}
		if mode == "private-field" {
			fmt.Fprintf(os.Stdout, "[{\"role\":\"host\",\"thread\":%d,\"complete\":false,\"reason\":\"api-unavailable\",\"object_name\":\"private-native-name\"}]", value.Thread)
		} else {
			json.NewEncoder(os.Stdout).Encode([]nativeWaitObservation{value})
		}
		os.Exit(0)
	case "normal", "native":
		query := queryNativeWait
		if mode == "normal" {
			query = func(thread uint32) nativeWaitObservation {
				return nativeWaitObservation{Thread: thread, Complete: true, Nodes: []nativeWaitItem{{Kind: 8, Status: 2, PID: 123, Thread: thread}}}
			}
		}
		os.Exit(writeNativeWaitQuery(os.Stdin, os.Stdout, query))
	}
	os.Exit(97)
}

func nativeWaitTestCapture(mode string) nativeWaitCapture {
	return func(ctx context.Context, input []byte) (process.CaptureResult, error) {
		executable, err := os.Executable()
		if err != nil {
			return process.CaptureResult{ExitCode: -1}, err
		}
		return process.Capture(ctx, process.Spec{Executable: executable, Args: []string{"-test.run=^TestNativeWaitObserverHelperProcess$"}, Env: append(os.Environ(), "INSONIC_NATIVE_WAIT_HELPER_MODE="+mode), Input: bytes.NewReader(input), MaxOutput: nativeWaitOutputLimit})
	}
}

func assertNativeWaitContainment(t *testing.T, mode, reason string) {
	t.Helper()
	var out bytes.Buffer
	var rows []nativeWaitObservation
	started := time.Now()
	writeNativeStartupProgress(&out, func() any {
		if !bytes.Contains(out.Bytes(), []byte(`"kind":"native-startup-progress"`)) {
			t.Fatal("primary record was not written before starting native helper")
		}
		rows = collectNativeWaits([]nativeWaitTarget{{Role: "host", Thread: 42}}, nativeWaitTestCapture(mode), 500*time.Millisecond)
		return rows
	})
	if elapsed := time.Since(started); elapsed > 1500*time.Millisecond {
		t.Fatalf("optional native work exceeded its bounded cleanup allowance: %s", elapsed)
	}
	if len(rows) != 1 || rows[0].Role != "host" || rows[0].Thread != 42 || rows[0].Complete || rows[0].Reason != reason {
		t.Fatalf("failed native helper was accepted: %+v", rows)
	}
	if mode == "crash" && rows[0].ObserverExitCode != int(uint32(0xc0000005)) {
		t.Fatalf("native failure code lost: %+v", rows[0])
	}
	lines := bytes.Split(bytes.TrimSpace(out.Bytes()), []byte{'\n'})
	if len(lines) != 2 || bytes.Contains(out.Bytes(), []byte("private-native")) {
		t.Fatalf("diagnostic failure erased primary evidence or exposed child output: %s", out.Bytes())
	}
	for index, kind := range []string{"native-startup-progress", "native-startup-waits"} {
		var record map[string]any
		if json.Unmarshal(lines[index], &record) != nil || record["kind"] != kind {
			t.Fatal("diagnostic record order changed", out.String())
		}
	}
}

func TestNativeWaitObservationCannotDelayFailedCaller(t *testing.T) {
	assertNativeWaitContainment(t, "timeout", "observation-deadline")
}

func TestNativeWaitHelperFaultsCannotErasePrimaryProgress(t *testing.T) {
	for _, tc := range []struct{ mode, reason string }{
		{"crash", "observer-failed"},
		{"malformed", "observer-output-invalid"},
		{"truncated", "observer-output-limit"},
		{"wrong-target", "observer-output-invalid"},
		{"too-many-nodes", "observer-output-invalid"},
		{"private-field", "observer-output-invalid"},
	} {
		t.Run(tc.mode, func(t *testing.T) { assertNativeWaitContainment(t, tc.mode, tc.reason) })
	}
}

func TestNativeWaitHelperReturnsOnlyRequestedBoundedMetadata(t *testing.T) {
	targets := []nativeWaitTarget{{Role: "host", Thread: 42}, {Role: "browser", Thread: 43}}
	rows := collectNativeWaits(targets, nativeWaitTestCapture("normal"), 500*time.Millisecond)
	if len(rows) != len(targets) {
		t.Fatal("target result count differs", rows)
	}
	for index, row := range rows {
		if !row.Complete || row.Role != targets[index].Role || row.Thread != targets[index].Thread || len(row.Nodes) != 1 || row.Nodes[0].PID != 123 || row.Nodes[0].Thread != targets[index].Thread {
			t.Fatalf("metadata changed across helper boundary: %+v", rows)
		}
	}
}

func TestNativeWaitHelperRejectsUnboundedOrAmbiguousRequests(t *testing.T) {
	for _, input := range []string{
		`[]`, `[{"role":"host","thread":0}]`, `[{"role":"browser","thread":42}]`,
		`[{"role":"host","thread":42},{"role":"browser","thread":42}]`,
		`[{"role":"host","thread":42},{"role":"browser","thread":43},{"role":"browser","thread":44}]`,
		`[{"role":"host","thread":42,"private":"input"}]`,
		`[{"role":"host","thread":42}] {}`, strings.Repeat(" ", 1025),
	} {
		var output bytes.Buffer
		called := false
		code := writeNativeWaitQuery(strings.NewReader(input), &output, func(thread uint32) nativeWaitObservation { called = true; return nativeWaitObservation{} })
		if code == 0 || called || output.Len() != 0 {
			t.Fatal("invalid request reached native API or emitted output")
		}
	}
}

func TestNativeWaitQueryReportsCurrentOwnedThreadOrUnavailability(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	thread := windows.GetCurrentThreadId()
	rows := collectNativeWaits([]nativeWaitTarget{{Role: "host", Thread: thread}}, nativeWaitTestCapture("native"), 500*time.Millisecond)
	if len(rows) != 1 || rows[0].Thread != thread {
		t.Fatal("native wait query lost target identity", rows)
	}
	value := rows[0]
	if value.Complete {
		if len(value.Nodes) == 0 || value.Nodes[0].Kind != 8 || value.Nodes[0].PID != uint32(os.Getpid()) || value.Nodes[0].Thread != thread {
			t.Fatal("native wait ABI did not recover the actual owned thread", value)
		}
	} else if value.Reason == "" && len(value.Nodes) == 0 {
		t.Fatal("native wait unavailable without a reason", value)
	}
	t.Logf("isolated native wait result: complete=%t reason=%s error=%d nodes=%d", value.Complete, value.Reason, value.Error, len(value.Nodes))
}
