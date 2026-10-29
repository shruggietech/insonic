// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

type capacityFaultStore struct {
	artifact.Store
	kind      string
	targetID  string
	blocked   bool
	uncertain bool
	calls     int
}

func (s *capacityFaultStore) PublishImmutable(ctx context.Context, p catalog.Publication, source *os.File, save func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error) {
	if s.blocked && p.Kind == s.kind && p.ID == s.targetID {
		s.calls++
		if s.uncertain {
			next, e := s.Store.PublishImmutable(ctx, p, source, save)
			if e != nil {
				return next, e
			}
			p = next
		}
		return p, contracts.Fail("insufficient_space")
	}
	return s.Store.PublishImmutable(ctx, p, source, save)
}
func (s *capacityFaultStore) Stat(ctx context.Context, p catalog.Publication) (artifact.Object, error) {
	if s.blocked && p.Kind == s.kind && p.ID == s.targetID {
		return artifact.Object{}, contracts.Fail("insufficient_space")
	}
	return s.Store.Stat(ctx, p)
}

func TestNativeStorageFailuresPreserveAcceptedAuthorityAndRecover(t *testing.T) {
	for _, kind := range []string{"canonical-audio", "media-metadata-report"} {
		for _, uncertain := range []bool{false, true} {
			t.Run(kind+map[bool]string{false: "-before-publication", true: "-uncertain-complete"}[uncertain], func(t *testing.T) {
				s, db := libraryFixture(t)
				s.legacyFixture = false
				s.Tools = nativeTools(t)
				ctx := context.Background()
				sentinelSource := wavFixture(t)
				first := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: sentinelSource}}})
				value, e := s.Execute(ctx, first)
				if e != nil || value.(ImportResult).Items[0].State != "admitted" {
					t.Fatal("sentinel failed", value, e)
				}
				sentinel, e := db.Library(ctx, value.(ImportResult).Items[0].MediaID)
				if e != nil {
					t.Fatal(e)
				}
				sentinelBytes, _ := json.Marshal(sentinel)
				source := filepath.Join(t.TempDir(), "new original.wav")
				raw, e := os.ReadFile(sentinelSource)
				if e != nil {
					t.Fatal(e)
				}
				raw[len(raw)-1] ^= 1
				if e = os.WriteFile(source, raw, 0600); e != nil {
					t.Fatal(e)
				}
				work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", OriginatedAt: "2020-01-02T03:04:05.123456789Z"}, Items: []Item{{Source: source}}})
				label := "canonical-0"
				if kind == "media-metadata-report" {
					label = "canonical-report-0"
				}
				fault := &capacityFaultStore{Store: s.Artifacts.Store, kind: kind, targetID: DerivedID(work.ID, label), blocked: true, uncertain: uncertain}
				s.Artifacts.Store = fault
				result, e := s.Execute(ctx, work)
				if e != nil {
					t.Fatal(e)
				}
				failed := result.(ImportResult)
				if !failed.Partial || failed.Items[0].State != "failed" || failed.Items[0].Error != "insufficient_space" || fault.calls < 1 {
					t.Fatal("failure was not observed", failed)
				}
				entries, e := db.Libraries(ctx)
				if e != nil || len(entries) != 1 {
					t.Fatal("incomplete import admitted", entries, e)
				}
				after, e := db.Library(ctx, sentinel.ID)
				encoded, _ := json.Marshal(after)
				if e != nil || !bytes.Equal(encoded, sentinelBytes) || s.Artifacts.Verify(ctx, *sentinel.OriginalPublicationID) != nil {
					t.Fatal("accepted authority changed", e)
				}
				original, e := os.ReadFile(source)
				if e != nil || !bytes.Equal(original, raw) {
					t.Fatal("source changed", e)
				}
				pending, e := db.Publication(ctx, DerivedID(work.ID, label))
				wantState := map[bool]string{false: "aborted", true: "pending"}[uncertain]
				if e != nil || pending.State != wantState {
					t.Fatal("failure disposition differs", pending, e)
				}
				if uncertain {
					if _, e = os.Stat(filepath.Join(s.Artifacts.Workspace.Control, "artifact-scratch", pending.ID+".source")); e != nil {
						t.Fatal("uncertain recovery source lost", e)
					}
				}
				fault.blocked = false
				if uncertain {
					if p, e := s.Artifacts.Reconcile(ctx, pending.ID); e != nil || p.State != "available" {
						t.Fatal("uncertain publication did not reconcile", p, e)
					}
				}
				retry := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC", OriginatedAt: "2020-01-02T03:04:05.123456789Z"}, Items: []Item{{Source: source}}})
				result, e = s.Execute(ctx, retry)
				if e != nil || result.(ImportResult).Items[0].State != "admitted" {
					t.Fatal("ordinary new import did not recover", result, e)
				}
				entry, e := db.Library(ctx, result.(ImportResult).Items[0].MediaID)
				if e != nil || s.Artifacts.Verify(ctx, *entry.OriginalPublicationID) != nil {
					t.Fatal("accepted bytes invalid", e)
				}
				if d := decodeDates(entry.Dates).Selected; d == nil || d.Resolved == nil || d.Resolved.UnixNS != 1577934245123456789 {
					t.Fatal("owner date lost")
				}
				ids, e := catalog.PublicationIDs(entry.ReportPublicationIDs)
				if e != nil || len(ids) != 1 {
					t.Fatal("report authority missing", e)
				}
				report, e := db.Publication(ctx, ids[0])
				if e != nil || s.Artifacts.Verify(ctx, report.ID) != nil {
					t.Fatal("report bytes invalid", e)
				}
				m, e := s.Artifacts.MaterializeBound(ctx, report.ID, report.Size)
				if e != nil {
					t.Fatal(e)
				}
				data, readError := os.ReadFile(m.Path)
				if e = s.Artifacts.Release(ctx, report.ID, m.Lease.ID); e != nil {
					t.Fatal(e)
				}
				var bundle captureBundle
				wantDigest, wantSize := identityBytes(raw)
				if readError != nil || strict(data, &bundle) != nil || bundle.SourceDigest != wantDigest || bundle.SourceSize != wantSize || !bytes.Equal(entry.Metadata, marshal(bundle.Metadata)) {
					t.Fatal("source/capture provenance differs", readError)
				}
				entries, e = db.Libraries(ctx)
				if e != nil || len(entries) != 2 {
					t.Fatal("recovery admission count differs", e)
				}
				replay, e := s.Execute(ctx, retry)
				if e != nil || replay.(ImportResult).Items[0].MediaID != entry.ID {
					t.Fatal("accepted replay differs", replay, e)
				}
			})
		}
	}
}
