//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/shruggietech/insonic/internal/process"
	"golang.org/x/sys/windows"
)

// Windows amd64 WAITCHAIN_NODE_INFO: two DWORD enums and a 272-byte union.
// Lock-object names are deliberately never decoded or retained in output.
type nativeWaitNode struct {
	Kind, Status uint32
	Data         [34]uint64
}
type nativeWaitItem struct {
	Kind   uint32 `json:"kind"`
	Status uint32 `json:"status"`
	PID    uint32 `json:"pid,omitempty"`
	Thread uint32 `json:"thread,omitempty"`
}
type nativeWaitObservation struct {
	Role             string           `json:"role"`
	Thread           uint32           `json:"thread"`
	Complete         bool             `json:"complete"`
	Cycle            bool             `json:"cycle"`
	Error            uint32           `json:"error,omitempty"`
	Reason           string           `json:"reason,omitempty"`
	Nodes            []nativeWaitItem `json:"nodes,omitempty"`
	ObserverExitCode int              `json:"observer_exit_code,omitempty"`
}

func nativeWaitItems(nodes []nativeWaitNode) []nativeWaitItem {
	result := make([]nativeWaitItem, 0, min(len(nodes), 16))
	for _, node := range nodes[:min(len(nodes), 16)] {
		item := nativeWaitItem{Kind: node.Kind, Status: node.Status}
		if node.Kind == 8 {
			item.PID = uint32(node.Data[0])
			item.Thread = uint32(node.Data[0] >> 32)
		}
		result = append(result, item)
	}
	return result
}
func queryNativeWait(thread uint32) nativeWaitObservation {
	result := nativeWaitObservation{Thread: thread}
	dll := windows.NewLazySystemDLL("advapi32.dll")
	open, query, closeSession := dll.NewProc("OpenThreadWaitChainSession"), dll.NewProc("GetThreadWaitChain"), dll.NewProc("CloseThreadWaitChainSession")
	for _, proc := range []*windows.LazyProc{open, query, closeSession} {
		if proc.Find() != nil {
			result.Reason = "api-unavailable"
			return result
		}
	}
	session, _, nativeError := open.Call(0, 0)
	if session == 0 {
		result.Reason = "session-unavailable"
		if code, ok := nativeError.(syscall.Errno); ok {
			result.Error = uint32(code)
		}
		return result
	}
	defer closeSession.Call(session)
	var nodes [16]nativeWaitNode
	count, cycle := uint32(len(nodes)), int32(0)
	var pin runtime.Pinner
	pin.Pin(&nodes)
	pin.Pin(&count)
	pin.Pin(&cycle)
	defer pin.Unpin()
	ok, _, nativeError := query.Call(session, 0, 1, uintptr(thread), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&nodes[0])), uintptr(unsafe.Pointer(&cycle)))
	if ok == 0 {
		if code, found := nativeError.(syscall.Errno); found {
			result.Error = uint32(code)
		}
		result.Reason = "query-unavailable"
	}
	result.Complete = ok != 0 && count <= uint32(len(nodes))
	result.Cycle = cycle != 0
	if ok != 0 || result.Error == uint32(windows.ERROR_MORE_DATA) || result.Error == uint32(windows.ERROR_TOO_MANY_THREADS) {
		result.Nodes = nativeWaitItems(nodes[:min(int(count), len(nodes))])
	}
	for _, node := range result.Nodes {
		if node.Kind == 10 || node.Status == 1 || node.Status == 4 || node.Status == 5 || node.Status == 9 || node.Status == 10 {
			result.Complete = false
		}
	}
	return result
}

const nativeWaitOutputLimit = 8 << 10

type nativeWaitTarget struct {
	Role   string `json:"role"`
	Thread uint32 `json:"thread"`
}

type nativeWaitCapture func(context.Context, []byte) (process.CaptureResult, error)

func validNativeWaitTargets(targets []nativeWaitTarget) bool {
	if len(targets) < 1 || len(targets) > 2 || targets[0].Role != "host" || targets[0].Thread == 0 {
		return false
	}
	return len(targets) == 1 || targets[1].Role == "browser" && targets[1].Thread != 0 && targets[1].Thread != targets[0].Thread
}

func decodeNativeWaitJSON(input io.Reader, limit int64, target any) bool {
	raw, err := io.ReadAll(io.LimitReader(input, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return false
	}
	var extra any
	return decoder.Decode(&extra) == io.EOF
}

func validNativeWaitResult(value nativeWaitObservation, target nativeWaitTarget) bool {
	if value.Role != target.Role || value.Thread != target.Thread || value.ObserverExitCode != 0 || len(value.Nodes) > 16 {
		return false
	}
	switch value.Reason {
	case "", "api-unavailable", "session-unavailable", "query-unavailable":
	default:
		return false
	}
	if value.Complete && (value.Reason != "" || value.Error != 0 || len(value.Nodes) == 0) {
		return false
	}
	if !value.Complete && value.Reason == "" && len(value.Nodes) == 0 {
		return false
	}
	for _, node := range value.Nodes {
		if node.Kind < 1 || node.Kind > 12 || node.Status < 1 || node.Status > 10 || node.Kind != 8 && (node.PID != 0 || node.Thread != 0) {
			return false
		}
	}
	return true
}

// This mode never starts Wails or native input. Native WCT faults and stuck
// synchronous queries remain inside the supervisor-owned hidden process.
func writeNativeWaitQuery(input io.Reader, output io.Writer, query func(uint32) nativeWaitObservation) int {
	var targets []nativeWaitTarget
	if !decodeNativeWaitJSON(input, 1024, &targets) || !validNativeWaitTargets(targets) {
		return 1
	}
	rows := make([]nativeWaitObservation, 0, len(targets))
	for _, target := range targets {
		value := query(target.Thread)
		value.Role = target.Role
		if !validNativeWaitResult(value, target) {
			return 1
		}
		rows = append(rows, value)
	}
	if json.NewEncoder(output).Encode(rows) != nil {
		return 1
	}
	return 0
}

func qualifyNativeWait() int { return writeNativeWaitQuery(os.Stdin, os.Stdout, queryNativeWait) }

func captureNativeWait(ctx context.Context, input []byte) (process.CaptureResult, error) {
	executable, err := os.Executable()
	if err != nil {
		return process.CaptureResult{ExitCode: -1}, err
	}
	return process.Capture(ctx, process.Spec{Executable: executable, Args: []string{"--native-wait-query"}, Input: bytes.NewReader(input), MaxOutput: nativeWaitOutputLimit})
}

func unavailableNativeWaits(targets []nativeWaitTarget, reason string, exitCode int) []nativeWaitObservation {
	rows := make([]nativeWaitObservation, 0, len(targets))
	for _, target := range targets {
		rows = append(rows, nativeWaitObservation{Role: target.Role, Thread: target.Thread, Reason: reason, ObserverExitCode: exitCode})
	}
	return rows
}

func collectNativeWaits(targets []nativeWaitTarget, capture nativeWaitCapture, limit time.Duration) []nativeWaitObservation {
	if !validNativeWaitTargets(targets) {
		return nil
	}
	input, _ := json.Marshal(targets)
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	report, err := capture(ctx, input)
	if report.Truncated {
		return unavailableNativeWaits(targets, "observer-output-limit", report.ExitCode)
	}
	if ctx.Err() != nil {
		return unavailableNativeWaits(targets, "observation-deadline", report.ExitCode)
	}
	if err != nil || report.ExitCode != 0 {
		return unavailableNativeWaits(targets, "observer-failed", report.ExitCode)
	}
	var rows []nativeWaitObservation
	if !decodeNativeWaitJSON(bytes.NewReader(report.Stdout), nativeWaitOutputLimit, &rows) || len(rows) != len(targets) {
		return unavailableNativeWaits(targets, "observer-output-invalid", report.ExitCode)
	}
	for i, row := range rows {
		if !validNativeWaitResult(row, targets[i]) {
			return unavailableNativeWaits(targets, "observer-output-invalid", report.ExitCode)
		}
	}
	return rows
}

func nativeStartupWaitSnapshot() any {
	hwnd := exitWinCall("GetForegroundWindow")
	if requireNativeInputTarget(hwnd) != nil {
		return nil
	}
	thread := exitWinCall("GetWindowThreadProcessId", hwnd, 0)
	targets := []nativeWaitTarget{{Role: "host", Thread: uint32(thread)}}
	info := exitGUIThreadInfo{Size: uint32(unsafe.Sizeof(exitGUIThreadInfo{}))}
	if exitWinCall("GetGUIThreadInfo", thread, uintptr(unsafe.Pointer(&info))) != 0 && info.Focus != 0 && exitWinCall("IsChild", hwnd, info.Focus) != 0 {
		browser := uint32(exitWinCall("GetWindowThreadProcessId", info.Focus, 0))
		if browser != 0 && browser != uint32(thread) {
			targets = append(targets, nativeWaitTarget{Role: "browser", Thread: browser})
		}
	}
	return collectNativeWaits(targets, captureNativeWait, 500*time.Millisecond)
}
