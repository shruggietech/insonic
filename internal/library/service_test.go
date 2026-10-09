// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func libraryFixture(t *testing.T) (*Service, catalog.Catalog) {
	t.Helper()
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "library")
	if e != nil {
		t.Fatal(e)
	}
	db, e := catalog.OpenWorkspace(ctx, w, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if e = db.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	a, e := artifact.NewService(ctx, w, db, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { a.Close() })
	a.Grace = 0
	s := NewService(a, db, nil, Tools{})
	s.legacyFixture = true
	return s, db
}
func claimLibraryWork(t *testing.T, db catalog.Catalog, kind string, payload any) catalog.Work {
	t.Helper()
	ctx := context.Background()
	w, e := db.EnqueueWork(ctx, contracts.ID(), kind, marshal(payload))
	if e != nil {
		t.Fatal(e)
	}
	w, e = db.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	return w
}
func TestAdmissionReferenceDedupRelocationAndRefresh(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	source := filepath.Join(t.TempDir(), "source.wav")
	os.WriteFile(source, []byte("RIFF\x00\x00\x00\x00WAVEfmt original"), 0600)
	copyMode := false
	req := ImportRequest{Defaults: Options{Copy: &copyMode, Timezone: "UTC", OriginatedOn: "2026-10-07"}, Items: []Item{{Source: source}}}
	work := claimLibraryWork(t, db, "media.import", req)
	value, e := s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	result := value.(ImportResult)
	if result.Items[0].State != "admitted" {
		t.Fatalf("result %+v", result)
	}
	entry, e := db.Library(ctx, result.Items[0].MediaID)
	if e != nil || entry.OriginalPublicationID != nil || entry.DurationUS != nil {
		t.Fatalf("reference %+v %v", entry, e)
	}
	second := claimLibraryWork(t, db, "media.import", req)
	value, e = s.Execute(ctx, second)
	if e != nil || value.(ImportResult).Items[0].MediaID != entry.ID {
		t.Fatal("repeat created duplicate entry")
	}
	moved := filepath.Join(filepath.Dir(source), "moved.wav")
	os.Rename(source, moved)
	entry, e = s.Relocate(ctx, contracts.ID(), entry.ID, entry.Revision, moved)
	if e != nil || entry.SourceLocator != moved {
		t.Fatalf("relocation %+v %v", entry, e)
	}
	oldReports, _ := catalog.PublicationIDs(entry.ReportPublicationIDs)
	work = claimLibraryWork(t, db, "media.refresh", RefreshRequest{MediaID: entry.ID})
	_, e = s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	current, e := db.Library(ctx, entry.ID)
	if e != nil {
		t.Fatal(e)
	}
	dates := decodeDates(current.Dates)
	if dates.Selected == nil || dates.Selected.Literal != "2026-10-07" {
		t.Fatal("owner date lost")
	}
	for _, id := range oldReports {
		p, e := db.Publication(ctx, id)
		if e != nil || p.State != "retired" {
			t.Fatalf("old report retained %+v %v", p, e)
		}
	}
	os.WriteFile(moved, []byte("changed"), 0600)
	shown, e := s.Show(ctx, entry.ID)
	if e != nil || shown.Availability != "changed" {
		t.Fatalf("external change %+v %v", shown, e)
	}
}
func TestPartialBatchKeepsUsableMedia(t *testing.T) {
	s, db := libraryFixture(t)
	source := filepath.Join(t.TempDir(), "source.wav")
	os.WriteFile(source, []byte("source"), 0600)
	r := ImportRequest{Defaults: Options{Timezone: "America/New_York"}, Items: []Item{{Source: source, Options: Options{OriginatedAt: "2026-11-01T01:30:00"}}, {Source: source + "-missing"}}}
	work := claimLibraryWork(t, db, "media.import", r)
	value, e := s.Execute(context.Background(), work)
	if e != nil {
		t.Fatal(e)
	}
	result := value.(ImportResult)
	if !result.Partial || result.Items[0].State != "admitted" || result.Items[0].DateState != "ambiguous" || result.Items[1].State != "failed" {
		data, _ := json.Marshal(result)
		t.Fatalf("partial %s", data)
	}
}
func TestRefreshPreservesApproximateOwnerBounds(t *testing.T) {
	s, db := libraryFixture(t)
	source := wavFixture(t)
	lower, _ := BoundInstant("2026-10-01T00:00:00Z")
	upper, _ := BoundInstant("2026-10-07T23:59:59.123456789Z")
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", OriginatedEarliest: lower, OriginatedLatest: upper}, Items: []Item{{Source: source}}})
	value, e := s.Execute(context.Background(), work)
	if e != nil {
		t.Fatal(e)
	}
	id := value.(ImportResult).Items[0].MediaID
	work = claimLibraryWork(t, db, "media.refresh", RefreshRequest{MediaID: id})
	if _, e = s.Execute(context.Background(), work); e != nil {
		t.Fatal(e)
	}
	entry, e := db.Library(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	selected := decodeDates(entry.Dates).Selected
	if selected == nil || selected.Resolved != nil || selected.Precision != "range" || selected.Lower.UnixNS != lower.UnixNS || selected.Upper.UnixNS != upper.UnixNS {
		t.Fatal("refresh dropped approximate owner bounds")
	}
}

func TestBatchBeyondOneHundredPreservesEveryCheckpointResult(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	source := wavFixture(t)
	r := ImportRequest{Defaults: Options{Timezone: "UTC", OriginatedOn: "2026-10-07"}}
	for range 125 {
		r.Items = append(r.Items, Item{Source: source, NewEntry: true})
	}
	work := claimLibraryWork(t, db, "media.import", r)
	value, e := s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	result := value.(ImportResult)
	if len(result.Items) != 125 {
		t.Fatalf("batch truncated to %d", len(result.Items))
	}
	encoded := marshal(result)
	if len(encoded) <= 16<<10 {
		t.Fatal("fixture did not exceed legacy result budget")
	}
	entries, e := db.Libraries(ctx)
	if e != nil || len(entries) != 125 {
		t.Fatalf("stored entries %d: %v", len(entries), e)
	}
	for _, item := range result.Items {
		if item.State != "admitted" {
			t.Fatal("usable item lost in large batch")
		}
	}
	// A full retry must reuse every deterministic entry, including items beyond
	// the default UI page, and preserve the exact ordered result.
	value, e = s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	if string(marshal(value)) != string(encoded) {
		t.Fatal("retry changed completed batch result")
	}
}

func TestDuplicateAdmissionChangesFoldAndRefreshPreservesChoice(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	source := wavFixture(t)
	options := Options{OriginatedAt: "2026-11-01T01:30:00", Timezone: "America/New_York", DSTFold: "earlier"}
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: options, Items: []Item{{Source: source}}})
	value, e := s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	id := value.(ImportResult).Items[0].MediaID
	options.DSTFold = "later"
	work = claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: options, Items: []Item{{Source: source}}})
	value, e = s.Execute(ctx, work)
	if e != nil || value.(ImportResult).Items[0].MediaID != id {
		t.Fatalf("duplicate correction failed: %v", e)
	}
	entry, e := db.Library(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	state := decodeDates(entry.Dates)
	if state.Selected.Resolved.ISO != "2026-11-01T06:30:00Z" || !state.Conflict {
		t.Fatal("later fold choice ignored")
	}
	work = claimLibraryWork(t, db, "media.refresh", RefreshRequest{MediaID: id})
	entry, e = s.Refresh(ctx, work, RefreshRequest{MediaID: id})
	if e != nil {
		t.Fatal(e)
	}
	state = decodeDates(entry.Dates)
	if state.Selected.Resolved.ISO != "2026-11-01T06:30:00Z" || !state.Conflict || len(state.Selected.Assumptions) < 1 || state.Selected.Assumptions[0] != "dst-fold-later" {
		t.Fatal("refresh erased owner fold choice/provenance")
	}
}
