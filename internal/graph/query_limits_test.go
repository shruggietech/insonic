// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

type limitEngine struct {
	rows   Rows
	native bool
	pages  []int
}

func (e *limitEngine) Close() {}
func (e *limitEngine) Begin(context.Context, bool) (session, error) {
	s := limitSession{e}
	if e.native {
		return nativeLimitSession{s}, nil
	}
	return s, nil
}

type limitSession struct{ e *limitEngine }

func (s limitSession) Commit(context.Context) error { return nil }
func (s limitSession) Rollback(context.Context)     {}
func (s limitSession) Query(_ context.Context, _ bool, q string, p map[string]any) (Rows, error) {
	if p["offset"] == nil {
		return s.e.rows, nil
	}
	if strings.Contains(q, "EvidenceLink") {
		return Rows{}, nil
	}
	offset, size := p["offset"].(int), p["limit"].(int)
	s.e.pages = append(s.e.pages, offset)
	// Model the transport budget forcing the page backoff before data is returned.
	if size > 500 {
		return nil, contracts.Fail("output_limit")
	}
	end := min(len(s.e.rows), offset+size)
	return s.e.rows[offset:end], nil
}

type nativeLimitSession struct{ limitSession }

func (s nativeLimitSession) Native(context.Context, string, string, map[string]any) (Rows, error) {
	return s.e.rows, nil
}
func TestNativeOutputBudgetSharedByRoutedAndDirectSessions(t *testing.T) {
	for _, native := range []bool{false, true} {
		for _, rows := range []Rows{make(Rows, 501), {{"value": strings.Repeat("x", (512<<10)+1)}}} {
			backend, dialect := "ladybugdb", "ladybug-cypher"
			text := "RETURN 1 AS value"
			if native {
				backend, dialect, text = "arcadedb", "arcade-sql", "SELECT 1 AS value"
			}
			a := &adapter{engine: &limitEngine{rows: rows, native: native}, backend: backend}
			_, e := a.Query(context.Background(), dialect, text, nil)
			if code, ok := e.(*contracts.Error); !ok || code.Code != "output_limit" {
				t.Fatalf("native=%v: %v", native, e)
			}
		}
	}
}
func TestReferenceSnapshotLargerThanWorkBudgetWithBoundedReads(t *testing.T) {
	nodes := make([]Node, 1400)
	rows := make(Rows, len(nodes))
	for i := range nodes {
		ref, _ := json.Marshal(map[string]any{"source_id": strings.Repeat("x", 6000)})
		nodes[i] = Node{ID: fmt.Sprint(i), Kind: "media", Reference: ref}
		rows[i] = map[string]any{"id": nodes[i].ID, "kind": "media", "reference": string(ref)}
	}
	claim := testClaim(contracts.ID(), contracts.ID(), 1, 1, Change{Kind: "refresh", Nodes: nodes})
	if len(claim.Document) <= contracts.MaxWorkPayload {
		t.Fatal("small fixture")
	}
	if _, e := validateChange(claim); e != nil {
		t.Fatal(e)
	}
	e := &limitEngine{rows: rows}
	a := &adapter{engine: e, backend: "ladybugdb"}
	refs, err := a.ReadRefs(context.Background(), claim.WorkspaceID)
	if err != nil || len(refs.Nodes) != len(nodes) {
		t.Fatal(len(refs.Nodes), err)
	}
	if len(e.pages) < 4 || e.pages[0] != 0 || e.pages[1] != 0 || e.pages[2] != 500 || e.pages[3] != 1000 {
		t.Fatal(e.pages)
	}
}
