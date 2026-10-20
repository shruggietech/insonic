//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"golang.org/x/sys/windows"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"
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
func TestNativeWaitObservationCannotDelayFailedCaller(t *testing.T) {
	release, finished := make(chan struct{}), make(chan struct{})
	defer func() { close(release); <-finished }()
	result := make(chan []nativeWaitObservation, 1)
	go func() {
		result <- collectNativeWaits([]nativeWaitObservation{{Role: "host", Thread: 42}}, func(thread uint32) nativeWaitObservation {
			defer close(finished)
			<-release
			return nativeWaitObservation{Thread: thread, Complete: true}
		}, 10*time.Millisecond)
	}()
	select {
	case rows := <-result:
		if len(rows) != 1 || rows[0].Complete || rows[0].Reason != "observation-deadline" {
			t.Fatal("pending native query conferred availability", rows)
		}
	case <-time.After(time.Second):
		t.Fatal("native query blocked failed caller")
	}
}

func TestNativeWaitQueryReportsCurrentOwnedThreadOrUnavailability(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	thread := windows.GetCurrentThreadId()
	rows := collectNativeWaits([]nativeWaitObservation{{Role: "host", Thread: thread}}, queryNativeWait, 500*time.Millisecond)
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
	t.Logf("native wait result: complete=%t reason=%s error=%d nodes=%d", value.Complete, value.Reason, value.Error, len(value.Nodes))
}
