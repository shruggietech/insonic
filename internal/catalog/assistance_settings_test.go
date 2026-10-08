// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestNamedSettingCASReplayAndRestore(t *testing.T) { namedSettingSuite(t, localStore) }
func namedSettingSuite(t *testing.T, create func(*testing.T, string) *Store) {
	ctx := context.Background()
	s := create(t, contracts.ID())
	raw, rev, e := s.NamedSetting(ctx, "query-assistance")
	if e != nil || rev != 0 || len(raw) != 0 {
		t.Fatal(raw, rev, e)
	}
	op := contracts.ID()
	value := json.RawMessage(`{"enabled":false}`)
	receipt, e := s.PutNamedSetting(ctx, op, "query-assistance", 0, value)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := s.PutNamedSetting(ctx, op, "query-assistance", 0, value)
	if e != nil || replay.Revision != receipt.Revision {
		t.Fatal(replay, e)
	}
	if _, e = s.PutNamedSetting(ctx, contracts.ID(), "query-assistance", 0, value); e == nil {
		t.Fatal("stale write accepted")
	}
	if _, e = s.PutNamedSetting(ctx, op, "query-assistance", 0, json.RawMessage(`{"enabled":true}`)); e == nil {
		t.Fatal("changed replay accepted")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	dest := create(t, s.workspace)
	if e = dest.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	got, next, e := dest.NamedSetting(ctx, "query-assistance")
	if e != nil || string(got) != string(value) || next != receipt.Revision {
		t.Fatal(string(got), next, e)
	}
}
