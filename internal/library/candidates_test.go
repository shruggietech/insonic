// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

func subtitleFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "supplied.srt")
	if e := os.WriteFile(path, []byte("1\n00:00:00,000 --> 00:00:00,010\nFixture\n"), 0600); e != nil {
		t.Fatal(e)
	}
	return path
}
func requireCandidateRetired(t *testing.T, s *Service, db catalog.Catalog, id string) {
	t.Helper()
	p, e := db.Publication(context.Background(), id)
	if e != nil || p.State != "retired" {
		t.Fatalf("unused candidate not retired %+v %v", p, e)
	}
	if _, e = s.Artifacts.Store.Stat(context.Background(), p); e == nil {
		t.Fatal("unused candidate bytes retained")
	}
}

type mutateSourceOnReport struct {
	artifact.Store
	source string
}

func (s *mutateSourceOnReport) PublishImmutable(ctx context.Context, p catalog.Publication, f *os.File, checkpoint func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error) {
	result, e := s.Store.PublishImmutable(ctx, p, f, checkpoint)
	if e == nil && p.Kind == "media-metadata-report" {
		if e = os.WriteFile(s.source, []byte("replacement-source-after-capture"), 0600); e != nil {
			return result, e
		}
	}
	return result, e
}
func TestFailedAdmissionRetiresOnlyItsUnusedSourceAndSubtitleCandidates(t *testing.T) {
	for _, failure := range []string{"extract", "commit", "changed-source"} {
		t.Run(failure, func(t *testing.T) {
			s, db := libraryFixture(t)
			ctx := context.Background()
			source := wavFixture(t)
			subtitle := subtitleFixture(t)
			work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: source, Subtitle: subtitle}}})
			switch failure {
			case "extract":
				s.Tools.Kind = "incompatible"
			case "commit":
				s.Catalog = &failLibraryCommit{Catalog: db, fail: true}
			case "changed-source":
				s.Artifacts.Store = &mutateSourceOnReport{Store: s.Artifacts.Store, source: source}
			}
			value, e := s.Execute(ctx, work)
			if e != nil || value.(ImportResult).Items[0].State != "failed" {
				t.Fatalf("injected failure did not fail admission: %v", e)
			}
			for _, label := range []string{"original-0", "subtitle-0"} {
				requireCandidateRetired(t, s, db, DerivedID(work.ID, label))
			}
			if failure != "extract" {
				requireCandidateRetired(t, s, db, DerivedID(work.ID, "report-0"))
			}
			for _, path := range []string{source, subtitle} {
				if _, e = os.Stat(path); e != nil {
					t.Fatal("caller-owned input was removed")
				}
			}
		})
	}
}

type unknownLibraryCommit struct{ catalog.Catalog }

func (s unknownLibraryCommit) CommitLibrary(ctx context.Context, w catalog.Work, e catalog.LibraryEntry) (catalog.LibraryEntry, error) {
	out, err := s.Catalog.CommitLibrary(ctx, w, e)
	if err == nil {
		return out, contracts.Fail("unavailable")
	}
	return out, err
}
func TestUnknownSuccessfulCommitPreservesSelectedOriginalSubtitleAndReport(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: wavFixture(t), Subtitle: subtitleFixture(t)}}})
	s.Catalog = unknownLibraryCommit{Catalog: db}
	value, e := s.Execute(ctx, work)
	if e != nil || value.(ImportResult).Items[0].State != "failed" {
		t.Fatal("unknown response fixture failed")
	}
	entry, e := db.Library(ctx, DerivedID(work.ID, "media-0"))
	if e != nil {
		t.Fatal(e)
	}
	reports, _ := catalog.PublicationIDs(entry.ReportPublicationIDs)
	for _, id := range append(reports, *entry.OriginalPublicationID, *entry.SubtitlePublicationID) {
		p, e := db.Publication(ctx, id)
		if e != nil || p.State != "available" {
			t.Fatal("selected candidate retired after uncertain commit")
		}
		if e = s.Artifacts.Verify(ctx, id); e != nil {
			t.Fatal(e)
		}
	}
	value, e = s.Execute(ctx, work)
	if e != nil || value.(ImportResult).Items[0].State != "admitted" {
		t.Fatal("committed entry not recovered")
	}
}

func TestSharedOriginalSurvivesAnotherEntryFailedCommit(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	source := wavFixture(t)
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: source}}})
	value, e := s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	entry, e := db.Library(ctx, value.(ImportResult).Items[0].MediaID)
	if e != nil {
		t.Fatal(e)
	}
	work = claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: source, NewEntry: true, Subtitle: subtitleFixture(t)}}})
	s.Catalog = &failLibraryCommit{Catalog: db, fail: true}
	value, e = s.Execute(ctx, work)
	if e != nil || value.(ImportResult).Items[0].State != "failed" {
		t.Fatal("shared-source failure not injected")
	}
	if e = s.Artifacts.Verify(ctx, *entry.OriginalPublicationID); e != nil {
		t.Fatal("existing shared original removed")
	}
	requireCandidateRetired(t, s, db, DerivedID(work.ID, "subtitle-0"))
	requireCandidateRetired(t, s, db, DerivedID(work.ID, "report-0"))
}

func TestUnusedSourceRetirementResumesAfterStorageRecovery(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: wavFixture(t), Subtitle: subtitleFixture(t)}}})
	s.Catalog = &failLibraryCommit{Catalog: db, fail: true}
	original := DerivedID(work.ID, "original-0")
	store := &transientReportDeletion{Store: s.Artifacts.Store, failures: 2, failID: original}
	s.Artifacts.Store = store
	value, e := s.Execute(ctx, work)
	if e != nil || value.(ImportResult).Items[0].State != "failed" {
		t.Fatal("commit failure not injected")
	}
	p, e := db.Publication(ctx, original)
	if e != nil || p.State != "retiring" {
		t.Fatal("original retirement not durable")
	}
	value, e = s.Execute(ctx, work)
	if e != nil || value.(ImportResult).Items[0].State != "failed" {
		t.Fatal("storage failure bypassed")
	}
	value, e = s.Execute(ctx, work)
	if e != nil || value.(ImportResult).Items[0].State != "admitted" {
		t.Fatalf("source retry did not recover: %v", e)
	}
	requireCandidateRetired(t, s, db, original)
	entry, e := db.Library(ctx, value.(ImportResult).Items[0].MediaID)
	if e != nil {
		t.Fatal(e)
	}
	if *entry.OriginalPublicationID == original {
		t.Fatal("retired original reselected")
	}
	if e = s.Artifacts.Verify(ctx, *entry.OriginalPublicationID); e != nil {
		t.Fatal("replacement original unusable")
	}
	if e = s.Artifacts.Verify(ctx, *entry.SubtitlePublicationID); e != nil {
		t.Fatal("replacement subtitle unusable")
	}
}
