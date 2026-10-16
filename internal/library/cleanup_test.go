// SPDX-License-Identifier: Apache-2.0
package library

import (
	"context"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

func refreshLeaseSuite(t *testing.T, s *Service, expire bool) {
	t.Helper()
	ctx := context.Background()
	db := s.Catalog
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Items: []Item{{Source: wavFixture(t)}}})
	value, err := s.Execute(ctx, work)
	if err != nil || value.(ImportResult).Items[0].State != "admitted" {
		t.Fatalf("admission: %v %v", value, err)
	}
	entry, err := db.Library(ctx, value.(ImportResult).Items[0].MediaID)
	if err != nil {
		t.Fatal(err)
	}
	old, err := catalog.PublicationIDs(entry.ReportPublicationIDs)
	if err != nil || len(old) == 0 {
		t.Fatal("missing reports", err)
	}
	lease, err := s.Artifacts.Materialize(ctx, old[0])
	if err != nil {
		t.Fatal(err)
	}
	defer s.Artifacts.Release(ctx, old[0], lease.Lease.ID)
	request := RefreshRequest{MediaID: entry.ID}
	work = claimLibraryWork(t, db, "media.refresh", request)
	current, err := s.Refresh(ctx, work, request)
	if err != nil {
		t.Fatal("accepted refresh failed under a live report lease", err)
	}
	if current.Revision <= entry.Revision || string(current.ReportPublicationIDs) == string(entry.ReportPublicationIDs) {
		t.Fatal("refresh did not select replacement reports")
	}
	pub, err := db.Publication(ctx, old[0])
	if err != nil || pub.State != "available" {
		t.Fatal("leased old report retired", err, pub.State)
	}
	if _, err = s.Artifacts.Store.Stat(ctx, pub); err != nil {
		t.Fatal("leased old report bytes missing", err)
	}
	replay, err := s.Refresh(ctx, work, request)
	if err != nil || replay.Revision != current.Revision || string(replay.ReportPublicationIDs) != string(current.ReportPublicationIDs) {
		t.Fatal("accepted retry changed authority or failed", replay.Revision, err)
	}
	// Reconstruct the service from the persisted catalog, as runtime startup does.
	reopened := NewService(s.Artifacts, db, s.Secrets, s.Tools)
	if expire {
		if _, err = db.ArtifactLease(ctx, old[0], lease.Lease.ID, lease.Lease.Owner, time.Nanosecond, false); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	} else {
		if err = s.Artifacts.Release(ctx, old[0], lease.Lease.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err = reopened.Cleanup(ctx, entry.ID); err != nil {
		t.Fatal("deferred cleanup did not reconcile", err)
	}
	pub, err = db.Publication(ctx, old[0])
	if err != nil || pub.State != "retired" {
		t.Fatal("old report not retired after release", err)
	}
	if _, err = s.Artifacts.Store.Stat(ctx, pub); err == nil {
		t.Fatal("obsolete report bytes retained")
	}
	final, err := db.Library(ctx, entry.ID)
	if err != nil || final.Revision != current.Revision {
		t.Fatal("cleanup changed accepted revision", err)
	}
	if final.OriginalPublicationID != nil {
		if err = s.Artifacts.Verify(ctx, *final.OriginalPublicationID); err != nil {
			t.Fatal("cleanup retired current canonical audio", err)
		}
	}
}

func TestRefreshDefersLeasedReportsAndReconcilesAcceptedRetry(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(map[bool]string{false: "release", true: "expiry"}[expire], func(t *testing.T) {
			s, _ := libraryFixture(t)
			refreshLeaseSuite(t, s, expire)
		})
	}
}

type cleanupFailureCatalog struct {
	catalog.Catalog
	rows        []catalog.Cleanup
	failureID   string
	failureCode string
}

func (db cleanupFailureCatalog) Cleanups(context.Context) ([]catalog.Cleanup, error) {
	return db.rows, nil
}
func (db cleanupFailureCatalog) Publication(ctx context.Context, id string) (catalog.Publication, error) {
	if id == db.failureID {
		return catalog.Publication{}, contracts.Fail(db.failureCode)
	}
	return db.Catalog.Publication(ctx, id)
}

func TestCleanupDoesNotHideRealFailureAfterDeferredLease(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Items: []Item{{Source: wavFixture(t)}}})
	value, err := s.Execute(ctx, work)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := db.Library(ctx, value.(ImportResult).Items[0].MediaID)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := catalog.PublicationIDs(entry.ReportPublicationIDs)
	if err != nil {
		t.Fatal(err)
	}
	lease, err := s.Artifacts.Materialize(ctx, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	defer s.Artifacts.Release(ctx, ids[0], lease.Lease.ID)
	failureID := contracts.ID()
	for _, failureCode := range []string{"unavailable", "conflict"} {
		for _, reverse := range []bool{false, true} {
			rows := []catalog.Cleanup{{ID: ids[0], EntryID: entry.ID}, {ID: failureID, EntryID: entry.ID}}
			if reverse {
				rows[0], rows[1] = rows[1], rows[0]
			}
			s.Catalog = cleanupFailureCatalog{Catalog: db, failureID: failureID, failureCode: failureCode, rows: rows}
			if err = s.Cleanup(ctx, entry.ID); code(err) != failureCode {
				t.Fatal("deferred conflict hid a real cleanup failure", err)
			}
			if err = s.acceptedCleanup(ctx, entry.ID); code(err) != failureCode {
				t.Fatal("accepted cleanup hid a real failure", err)
			}
		}
	}
}

type failingRetirementStore struct {
	artifact.Store
	fail bool
}

func (s *failingRetirementStore) DeleteUnreferenced(ctx context.Context, pub catalog.Publication) error {
	if s.fail {
		return contracts.Fail("unavailable")
	}
	return s.Store.DeleteUnreferenced(ctx, pub)
}

func TestRefreshStorageFailureRetainsAcceptedReportForRetry(t *testing.T) {
	s, db := libraryFixture(t)
	ctx := context.Background()
	work := claimLibraryWork(t, db, "media.import", ImportRequest{Items: []Item{{Source: wavFixture(t)}}})
	value, err := s.Execute(ctx, work)
	if err != nil {
		t.Fatal(err)
	}
	id := value.(ImportResult).Items[0].MediaID
	store := &failingRetirementStore{Store: s.Artifacts.Store, fail: true}
	s.Artifacts.Store = store
	request := RefreshRequest{MediaID: id}
	work = claimLibraryWork(t, db, "media.refresh", request)
	accepted, err := s.Refresh(ctx, work, request)
	if code(err) != "unavailable" {
		t.Fatal("real storage failure hidden", err)
	}
	current, err := db.Library(ctx, id)
	if err != nil || current.Revision != accepted.Revision {
		t.Fatal("accepted state lost", err)
	}
	store.fail = false
	replay, err := s.Refresh(ctx, work, request)
	if err != nil || replay.Revision != accepted.Revision {
		t.Fatal("retry duplicated acceptance", err)
	}
}
