//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"golang.org/x/sys/windows"
	"runtime"
	"syscall"
	"time"
	"unsafe"
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
	Role     string           `json:"role"`
	Thread   uint32           `json:"thread"`
	Complete bool             `json:"complete"`
	Cycle    bool             `json:"cycle"`
	Error    uint32           `json:"error,omitempty"`
	Reason   string           `json:"reason,omitempty"`
	Nodes    []nativeWaitItem `json:"nodes,omitempty"`
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
func collectNativeWaits(targets []nativeWaitObservation, query func(uint32) nativeWaitObservation, limit time.Duration) []nativeWaitObservation {
	targets = targets[:min(len(targets), 2)]
	results := append([]nativeWaitObservation(nil), targets...)
	type item struct {
		index int
		value nativeWaitObservation
	}
	done := make(chan item, len(targets))
	for index, target := range targets {
		go func() { value := query(target.Thread); value.Role = target.Role; done <- item{index, value} }()
	}
	timer := time.NewTimer(limit)
	defer timer.Stop()
	for received := 0; received < len(targets); received++ {
		select {
		case item := <-done:
			results[item.index] = item.value
		case <-timer.C:
			for i := range results {
				if results[i].Reason == "" && results[i].Nodes == nil && !results[i].Complete {
					results[i].Reason = "observation-deadline"
				}
			}
			return results
		}
	}
	return results
}
func nativeStartupWaitSnapshot() any {
	hwnd := exitWinCall("GetForegroundWindow")
	if requireNativeInputTarget(hwnd) != nil {
		return nil
	}
	thread := exitWinCall("GetWindowThreadProcessId", hwnd, 0)
	targets := []nativeWaitObservation{{Role: "host", Thread: uint32(thread)}}
	info := exitGUIThreadInfo{Size: uint32(unsafe.Sizeof(exitGUIThreadInfo{}))}
	if exitWinCall("GetGUIThreadInfo", thread, uintptr(unsafe.Pointer(&info))) != 0 && info.Focus != 0 && exitWinCall("IsChild", hwnd, info.Focus) != 0 {
		browser := uint32(exitWinCall("GetWindowThreadProcessId", info.Focus, 0))
		if browser != 0 && browser != uint32(thread) {
			targets = append(targets, nativeWaitObservation{Role: "browser", Thread: browser})
		}
	}
	return collectNativeWaits(targets, queryNativeWait, 500*time.Millisecond)
}
