// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestSingleRowReadDoesNotReserveSQLiteWriter(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	w, err := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"items":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	// Establish both WAL connections before the writer is held. The second
	// connection must observe committed state without requesting a write lock.
	s.db.SetMaxOpenConns(2)
	one, err := s.db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	two, err := s.db.Conn(ctx)
	if err != nil {
		one.Close()
		t.Fatal(err)
	}
	one.Close()
	two.Close()
	writer, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Rollback()
	if _, err = writer.ExecContext(ctx, "UPDATE "+domainNamed("Works").name+" SET phase=? WHERE workspace_id=? AND id=?", "uncommitted", s.workspace, w.ID); err != nil {
		t.Fatal(err)
	}
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	got, err := s.Work(readCtx, w.ID)
	if err != nil || got.ID != w.ID || got.State != "pending" || got.Phase != "" {
		t.Fatal("single row read requested writer reservation or exposed uncommitted state", got, err)
	}
}

func TestSingleRowReadPreservesWorkspaceAndJSONAuthority(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	w, err := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"items":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	foreign := &Store{db: s.db, workspace: contracts.ID(), backend: s.backend}
	var failure *contracts.Error
	if _, err = foreign.Work(ctx, w.ID); !errors.As(err, &failure) || failure.Code != "not_found" {
		t.Fatal("single row read crossed workspace boundary", err)
	}
	if _, err = s.db.ExecContext(ctx, "UPDATE "+domainNamed("Works").name+" SET payload=? WHERE workspace_id=? AND id=?", `{"items":[],"items":[]}`, s.workspace, w.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Work(ctx, w.ID); !errors.As(err, &failure) || failure.Code != "invalid_request" {
		t.Fatal("single row read accepted ambiguous JSON", err)
	}
}
