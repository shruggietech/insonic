// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

type failLibraryCommit struct {
	catalog.Catalog
	fail bool
}

func (f *failLibraryCommit) CommitLibrary(ctx context.Context, work catalog.Work, entry catalog.LibraryEntry) (catalog.LibraryEntry, error) {
	if f.fail {
		f.fail = false
		return entry, contracts.Fail("operation_failed")
	}
	return f.Catalog.CommitLibrary(ctx, work, entry)
}

func TestFailedCommitRetiresComputedReportAndRetryAdmits(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	source := wavFixture(t)
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: source}}})
	s.Catalog = &failLibraryCommit{Catalog: db, fail: true}
	value, e := s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	if value.(ImportResult).Items[0].State != "failed" {
		t.Fatal("injected commit did not fail")
	}
	report := DerivedID(work.ID, "report-0")
	publication, e := db.Publication(ctx, report)
	if e != nil || publication.State != "retired" {
		t.Fatalf("failed report remains %+v %v", publication, e)
	}
	if _, e = s.Artifacts.Store.Stat(ctx, publication); e == nil {
		t.Fatal("failed report bytes remain")
	}
	value, e = s.Execute(ctx, work)
	if e != nil || value.(ImportResult).Items[0].State != "admitted" {
		t.Fatalf("retry failed: %v", e)
	}
}

func TestRecoveredCaptureCannotDescribeChangedSource(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	source := wavFixture(t)
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: source}}})
	directory := t.TempDir()
	stage, digest, size, e := s.Artifacts.Stage(ctx, source)
	if e != nil {
		t.Fatal(e)
	}
	stage.Close()
	defer os.Remove(stage.Name())
	bundle, e := s.capture(ctx, stage.Name(), source, Options{Timezone: "UTC"})
	if e != nil {
		t.Fatal(e)
	}
	if bundle.SourceDigest != digest || bundle.SourceSize != size {
		t.Fatal("capture not bound")
	}
	report := DerivedID(work.ID, "report-0")
	if e = s.publishBundle(ctx, report, bundle, directory); e != nil {
		t.Fatal(e)
	}
	// This is the observable state after process loss following publication and
	// before CommitLibrary. A replacement file must not inherit the old report.
	changed := filepath.Join(directory, "changed.wav")
	if e = os.WriteFile(changed, []byte("changed-source-bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	_, e = s.captureOrRecover(ctx, work, report, changed, source, Options{Timezone: "UTC"}, directory)
	if code(e) != "conflict" {
		t.Fatalf("mismatched recovered capture accepted: %v", e)
	}
	recovered, e := s.captureOrRecover(ctx, work, report, stage.Name(), source, Options{Timezone: "UTC"}, directory)
	if e != nil || recovered.SourceDigest != digest {
		t.Fatalf("same-byte crash recovery failed: %v", e)
	}
}

func TestCancelledDuplicateAdmissionCannotChangeOwnerDate(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	source := wavFixture(t)
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", OriginatedOn: "2026-10-01"}, Items: []Item{{Source: source}}})
	value, e := s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	entry, e := db.Library(ctx, value.(ImportResult).Items[0].MediaID)
	if e != nil {
		t.Fatal(e)
	}
	stale := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", OriginatedOn: "2026-10-07"}, Items: []Item{{Source: source}}})
	if _, e = db.CancelWork(ctx, contracts.ID(), stale.ID); e != nil {
		t.Fatal(e)
	}
	_, e = s.admit(ctx, stale, 0, Item{Source: source}, Options{Timezone: "UTC", OriginatedOn: "2026-10-07"}, newLibraryLookup([]catalog.LibraryEntry{entry}))
	if e == nil {
		t.Fatal("stale Work changed dates")
	}
	after, e := db.Library(ctx, entry.ID)
	if e != nil {
		t.Fatal(e)
	}
	if string(after.Dates) != string(entry.Dates) || after.Revision != entry.Revision {
		t.Fatal("cancelled duplicate mutated owner date")
	}
}
