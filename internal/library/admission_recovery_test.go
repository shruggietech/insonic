// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

type interruptedAdmission struct{ catalog.Catalog }

func (f interruptedAdmission) CommitAdmission(context.Context, catalog.Work, int, int64, catalog.LibraryEntry, *catalog.Recording, ...json.RawMessage) (catalog.LibraryEntry, *catalog.Recording, error) {
	panic("owned admission process loss")
}

func pendingNativeAdmission(t *testing.T) (*Service, catalog.Catalog, catalog.Work, string, catalog.Publication, catalog.Publication, captureBundle) {
	t.Helper()
	s, db := libraryFixture(t)
	s.legacyFixture = false
	s.Tools = nativeTools(t)
	source := wavFixture(t)
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", OriginatedAt: "2020-01-02T03:04:05.123456789Z"}, Items: []Item{{Source: source}}})
	s.Catalog = interruptedAdmission{db}
	func() {
		defer func() {
			if recovered := recover(); recovered != "owned admission process loss" {
				t.Fatalf("interruption not reached: %v", recovered)
			}
		}()
		_, _ = s.Execute(context.Background(), work)
	}()
	s.Catalog = db
	ctx := context.Background()
	canonical, err := db.Publication(ctx, DerivedID(work.ID, "canonical-0"))
	if err != nil || canonical.State != "available" {
		t.Fatal("canonical boundary not reached", err)
	}
	report, err := db.Publication(ctx, DerivedID(work.ID, "canonical-report-0"))
	if err != nil || report.State != "available" {
		t.Fatal("report boundary not reached", err)
	}
	materialized, err := s.Artifacts.MaterializeBound(ctx, report.ID, report.Size)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(materialized.Path)
	if release := s.Artifacts.Release(ctx, report.ID, materialized.Lease.ID); release != nil {
		t.Fatal(release)
	}
	var bundle captureBundle
	if err != nil || strict(data, &bundle) != nil {
		t.Fatal("pending report unreadable", err)
	}
	entries, err := db.Libraries(ctx)
	if err != nil || len(entries) != 0 {
		t.Fatal("fixture already admitted")
	}
	return s, db, work, source, canonical, report, bundle
}

func TestNativeInterruptedAdmissionPreservesPublishedCandidates(t *testing.T) {
	s, db, work, source, canonical, report, bundle := pendingNativeAdmission(t)
	ctx := context.Background()
	// A fresh extraction cannot satisfy this retry. Recovery must read the
	// already verified report and canonical bytes, retaining its capture instant.
	s.Tools = Tools{}
	result, err := s.Execute(ctx, work)
	if err != nil || result.(ImportResult).Items[0].State != "admitted" {
		t.Fatal("complete modern candidate was not recovered", result, err)
	}
	entry, err := db.Library(ctx, result.(ImportResult).Items[0].MediaID)
	if err != nil || entry.OriginalPublicationID == nil || *entry.OriginalPublicationID != canonical.ID || entry.Digest != canonical.Digest || !bytes.Equal(entry.Metadata, marshal(bundle.Metadata)) {
		t.Fatal("recovery replaced candidate bytes or original capture", err)
	}
	ids, err := catalog.PublicationIDs(entry.ReportPublicationIDs)
	if err != nil || len(ids) != 1 || ids[0] != report.ID {
		t.Fatal("original report identity replaced", err)
	}
	if selected := decodeDates(entry.Dates).Selected; selected == nil || selected.Literal != "2020-01-02T03:04:05.123456789Z" || selected.Basis != "owner" || selected.Precision != "instant" || selected.Resolved == nil || selected.Resolved.UnixNS != 1577934245123456789 {
		t.Fatal("entered date provenance lost")
	}
	for _, old := range []catalog.Publication{canonical, report} {
		current, err := db.Publication(ctx, old.ID)
		if err != nil || current.State != "available" || current.Digest != old.Digest || s.Artifacts.Verify(ctx, old.ID) != nil {
			t.Fatal("recovered publication changed", err)
		}
	}
	os.Remove(source)
	durable, err := db.Work(ctx, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err = s.Execute(ctx, durable)
	entries, readErr := db.Libraries(ctx)
	if err != nil || readErr != nil || result.(ImportResult).Items[0].MediaID != entry.ID || len(entries) != 1 {
		t.Fatal("recovery replay duplicated or reacquired", result, err, readErr)
	}
}

func TestNativeInterruptedAdmissionRejectsChangedSourceAndCorruptCandidates(t *testing.T) {
	for _, failure := range []string{"changed-source", "corrupt-report", "corrupt-canonical"} {
		t.Run(failure, func(t *testing.T) {
			s, db, work, source, canonical, report, _ := pendingNativeAdmission(t)
			path := source
			if failure != "changed-source" {
				publication := report
				if failure == "corrupt-canonical" {
					publication = canonical
				}
				path = filepath.Join(s.Artifacts.Workspace.Control, "artifacts", filepath.FromSlash(publication.Key))
			}
			data, err := os.ReadFile(path)
			if err != nil || len(data) == 0 {
				t.Fatal(err)
			}
			data[len(data)-1] ^= 1 // Equal length, different bytes.
			if err = os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			result, err := s.Execute(context.Background(), work)
			if err != nil || result.(ImportResult).Items[0].State != "failed" {
				t.Fatal("mismatched pending admission accepted", result, err)
			}
			if failure == "changed-source" && result.(ImportResult).Items[0].Error != "conflict" {
				t.Fatal("changed original was not classified", result)
			}
			entries, err := db.Libraries(context.Background())
			if err != nil || len(entries) != 0 {
				t.Fatal("failed recovery admitted a recording", err)
			}
		})
	}
}

func TestNativeInterruptedAdmissionPartialAndUnboundCandidatesRecompute(t *testing.T) {
	for _, unbound := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-report", true: "old-unbound-report"}[unbound], func(t *testing.T) {
			s, db := libraryFixture(t)
			s.legacyFixture, s.Tools = false, nativeTools(t)
			ctx := context.Background()
			source := wavFixture(t)
			options := Options{Timezone: "UTC"}
			work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: options, Items: []Item{{Source: source}}})
			directory := t.TempDir()
			bundle, err := s.capture(ctx, source, source, options)
			if err != nil {
				t.Fatal(err)
			}
			path, facts, err := s.canonicalize(ctx, source, directory, bundle)
			if err != nil {
				t.Fatal(err)
			}
			old, err := s.publish(ctx, DerivedID(work.ID, "canonical-0"), path, "canonical-audio")
			if err != nil {
				t.Fatal(err)
			}
			if unbound {
				bundle.Facts = facts
				if err = s.publishBundle(ctx, DerivedID(work.ID, "canonical-report-0"), bundle, directory); err != nil {
					t.Fatal(err)
				}
			}
			result, err := s.Execute(ctx, work)
			if err != nil || result.(ImportResult).Items[0].State != "admitted" {
				t.Fatal("partial candidate could not recompute", result, err)
			}
			entry, err := db.Library(ctx, result.(ImportResult).Items[0].MediaID)
			if err != nil || entry.OriginalPublicationID == nil || *entry.OriginalPublicationID == old.ID {
				t.Fatal("unproven canonical candidate was reused", err)
			}
			retired, err := db.Publication(ctx, old.ID)
			if err != nil || retired.State != "retired" {
				t.Fatal("superseded partial candidate was retained", err)
			}
		})
	}
}

func TestNativeInterruptedAdmissionStillRequiresCurrentWorkFence(t *testing.T) {
	s, db, work, _, canonical, report, _ := pendingNativeAdmission(t)
	ctx := context.Background()
	if _, err := db.CancelWork(ctx, contracts.ID(), work.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Execute(ctx, work); code(err) != "conflict" {
		t.Fatal("cancelled stale attempt recovered candidates", err)
	}
	for _, old := range []catalog.Publication{canonical, report} {
		current, err := db.Publication(ctx, old.ID)
		if err != nil || current.State != "available" || current.Digest != old.Digest {
			t.Fatal("stale attempt changed candidate authority", err)
		}
	}
}
