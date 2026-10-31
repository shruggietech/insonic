// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mattn/go-sqlite3"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestCatalogCapacityCausesRemainActionable(t *testing.T) {
	for _, cause := range []error{sqlite3.Error{Code: sqlite3.ErrFull}, &pgconn.PgError{Code: "53100", Message: "private-owner-secret"}} {
		got := contracts.SafeError(sanitize(fmt.Errorf("private-owner-secret: %w", cause)))
		if got.Code != "insufficient_space" || strings.Contains(got.Message, "private-owner-secret") {
			t.Fatalf("capacity cause lost or leaked: %+v", got)
		}
	}
}

func TestRestoreConfinedSQLiteCapacityRollbackAndRetry(t *testing.T) {
	ctx := context.Background()
	source := localStore(t, contracts.ID())
	value, _ := json.Marshal(strings.Repeat("bounded catalog authority", 32000))
	if _, e := source.PutNamedSetting(ctx, contracts.ID(), "capacity-fixture", 0, value); e != nil {
		t.Fatal(e)
	}
	snap, e := source.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	target := localStore(t, source.workspace)
	var pages int
	if e = target.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pages); e != nil {
		t.Fatal(e)
	}
	if _, e = target.db.ExecContext(ctx, fmt.Sprintf("PRAGMA max_page_count=%d", pages)); e != nil {
		t.Fatal(e)
	}
	if e = target.Restore(ctx, snap); contracts.SafeError(e).Code != "insufficient_space" {
		t.Fatalf("confined actual SQLite capacity failure lost: %v", e)
	}
	if e = target.Empty(ctx); e != nil {
		t.Fatal("failed restore admitted authority", e)
	}
	if _, e = target.db.ExecContext(ctx, "PRAGMA max_page_count=100000"); e != nil {
		t.Fatal(e)
	}
	if e = target.Restore(ctx, snap); e != nil {
		t.Fatal("identical retry", e)
	}
	got, _, e := target.NamedSetting(ctx, "capacity-fixture")
	if e != nil || string(got) != string(value) {
		t.Fatal("restored authority changed", e)
	}
	after, e := source.Export(ctx)
	if e != nil || after.Digest != snap.Digest {
		t.Fatal("source changed", e)
	}
}

func TestPortableRestoreConfinedCapacityRemainsFenced(t *testing.T) {
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "Portable capacity source")
	if e != nil {
		t.Fatal(e)
	}
	source := localStore(t, w.Config.WorkspaceID)
	if e = source.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	value, _ := json.Marshal(strings.Repeat("bounded portable authority", 32000))
	if _, e = source.PutNamedSetting(ctx, contracts.ID(), "capacity", 0, value); e != nil {
		t.Fatal(e)
	}
	snap, e := source.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	destination, e := workspace.Init(t.TempDir(), "Portable capacity destination")
	if e != nil {
		t.Fatal(e)
	}
	destination.Config.WorkspaceID = source.workspace
	target := localStore(t, source.workspace)
	var pages int
	if e = target.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pages); e != nil {
		t.Fatal(e)
	}
	if _, e = target.db.ExecContext(ctx, fmt.Sprintf("PRAGMA max_page_count=%d", pages)); e != nil {
		t.Fatal(e)
	}
	if _, e = target.RestorePortableFenced(ctx, snap, destination, nil); contracts.SafeError(e).Code != "insufficient_space" {
		t.Fatal("portable capacity cause", e)
	}
	if e = target.Empty(ctx); e != nil {
		t.Fatal("partial portable authority committed", e)
	}
	if _, e = target.db.ExecContext(ctx, "PRAGMA max_page_count=100000"); e != nil {
		t.Fatal(e)
	}
	lease, e := target.RestorePortableFenced(ctx, snap, destination, nil)
	if e != nil {
		t.Fatal("identical portable retry", e)
	}
	if e = target.EndTransfer(ctx, lease); e != nil {
		t.Fatal(e)
	}
	if _, e = target.PutNamedSetting(ctx, contracts.ID(), "must-stay-fenced", 0, json.RawMessage(`{}`)); contracts.SafeError(e).Code != "conflict" {
		t.Fatal("pending activation writable after lease release", e)
	}
	if ok, e := target.RestoredPortableWorkspace(ctx, snap.Digest, destination); e != nil || !ok {
		t.Fatal("retry authority missing", e)
	}
	after, e := source.Export(ctx)
	if e != nil || after.Digest != snap.Digest {
		t.Fatal("source changed", e)
	}
}
