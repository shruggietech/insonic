package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"strings"
	"testing"
	"time"
)

func TestOwnerProtocolAndDisconnectedWork(t *testing.T) {
	w, err := workspace.Init(t.TempDir(), "runtime")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, w, Options{Idle: time.Second, Ready: ready}) }()
	select {
	case <-ready:
	case <-time.After(3 * time.Second):
		t.Fatal("startup")
	}
	request := func(op string) contracts.Request {
		return contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: op}
	}
	first, err := Call(ctx, w, request("workspace.show"))
	if err != nil || first.Error != nil {
		t.Fatalf("call: %+v %v", first, err)
	}
	second, _ := Call(ctx, w, request("workspace.show"))
	if first.SessionID != second.SessionID {
		t.Fatal("multiple runtime owners")
	}
	wrong := request("doctor")
	wrong.WorkspaceID = contracts.ID()
	response, _ := Call(ctx, w, wrong)
	if response.Error == nil || response.Error.Code != "workspace_mismatch" {
		t.Fatal("wrong workspace accepted")
	}
	wrong = request("doctor")
	wrong.Version = "99"
	response, _ = Call(ctx, w, wrong)
	if response.Error == nil || response.Error.Code != "incompatible_version" {
		t.Fatal("protocol mismatch")
	}
	if err := Serve(ctx, w, Options{}); err == nil {
		t.Fatal("second owner started")
	}
	conn, err := dial(ctx, w)
	if err != nil {
		t.Fatal(err)
	}
	conn.Write([]byte(strings.Repeat("x", MaxFrame+2) + "\n"))
	data, err := bufio.NewReader(conn).ReadBytes('\n')
	conn.Close()
	if err != nil {
		t.Fatal(err)
	}
	var bad contracts.Response
	json.Unmarshal(data, &bad)
	if bad.Error == nil || bad.Error.Code != "invalid_request" {
		t.Fatal("oversized request")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown")
	}
	lock, err := w.Lock()
	if err != nil {
		t.Fatal("owner lock unreleased")
	}
	lock.Unlock()
}

func TestIdleExit(t *testing.T) {
	w, _ := workspace.Init(t.TempDir(), "idle")
	start := time.Now()
	if err := Serve(context.Background(), w, Options{Idle: 60 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("idle owner stayed alive")
	}
}

func TestWireRejectsUnknownFieldsAndTrailingDocuments(t *testing.T) {
	for _, data := range []string{`{"unknown":"fixture-secret"}`, `{"schema_version":"0.0.0"} {}`, `{`} {
		var req contracts.Request
		if err := decode([]byte(data), &req); err == nil || strings.Contains(err.Error(), "fixture-secret") {
			t.Fatal("invalid/untrusted wire accepted")
		}
	}
}
