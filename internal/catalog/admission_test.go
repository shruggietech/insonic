// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestElectedAudioReplacementAtomic(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	claim, entry := entryFixture(t, s)
	r := Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: 1, State: "untranscribed", Document: json.RawMessage(`null`), SourceMap: json.RawMessage(`{}`), Provenance: json.RawMessage(`{}`), Diagnostics: json.RawMessage(`[]`)}
	old, doc, e := s.CommitAdmission(ctx, claim, 0, 0, entry, &r)
	if e != nil {
		t.Fatal(e)
	}
	next, candidate := entryFixture(t, s)
	candidate.ID = old.ID
	candidate.Revision = old.Revision
	changed := r
	changed.SourceDigest = candidate.Digest
	if _, _, e = s.CommitAdmission(ctx, next, 0, doc.Revision, candidate, &changed); e == nil {
		t.Fatal("generic source mutation")
	}
	if _, _, e = s.CommitElectedAdmission(ctx, next, 0, doc.Revision+1, candidate, &changed, AdmissionElection{ReplaceAudio: true}); e == nil {
		t.Fatal("stale document accepted")
	}
	unchanged, _ := s.Library(ctx, old.ID)
	if unchanged.AssetID != old.AssetID {
		t.Fatal("failed mutation escaped")
	}
	got, newdoc, e := s.CommitElectedAdmission(ctx, next, 0, doc.Revision, candidate, &changed, AdmissionElection{ReplaceAudio: true})
	if e != nil {
		t.Fatal(e)
	}
	if got.ID != old.ID || got.AssetID == old.AssetID || newdoc.Revision != got.Revision || newdoc.State != "untranscribed" {
		t.Fatalf("replacement %+v %+v", got, newdoc)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("portable replacement", e)
	}
	stopped, candidate := entryFixture(t, s)
	candidate.ID = got.ID
	candidate.Revision = got.Revision
	if _, e = s.CancelWork(ctx, contracts.ID(), stopped.ID); e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.CommitElectedAdmission(ctx, stopped, 0, newdoc.Revision, candidate, newdoc, AdmissionElection{ReplaceAudio: true}); e == nil {
		t.Fatal("cancelled source publication")
	}
	actual, _ := s.Library(ctx, got.ID)
	if actual.Revision != got.Revision {
		t.Fatal("cancelled source changed current")
	}
}

func TestAdmissionAtomicReplayAndPortableProof(t *testing.T) {
	admissionSuite(t, localStore(t, contracts.ID()))
}
func admissionSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	claim, entry := entryFixture(t, s)
	r := Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: 1, State: "no-speech", Document: json.RawMessage("null"), SourceMap: json.RawMessage("{}"), Provenance: json.RawMessage("{}"), Diagnostics: json.RawMessage("[]")}
	bad := r
	bad.SourceDigest = "bad"
	if _, _, e := s.CommitAdmission(ctx, claim, 0, 0, entry, &bad); e == nil {
		t.Fatal("invalid candidate accepted")
	}
	if _, e := s.Library(ctx, entry.ID); e == nil {
		t.Fatal("partial library row accepted")
	}
	got, doc, e := s.CommitAdmission(ctx, claim, 0, 0, entry, &r)
	if e != nil {
		t.Fatal(e)
	}
	if doc == nil || got.Revision != doc.SourceRevision || got.Revision != doc.Revision {
		t.Fatal("admission was not one revision")
	}
	accepted, ok, e := s.AcceptedAdmissionWork(ctx, claim.ID, 0)
	if e != nil || !ok || len(accepted) == 0 {
		t.Fatal("accepted receipt missing", e)
	}
	replay, _, e := s.CommitAdmission(ctx, claim, 0, 0, entry, &r)
	if e != nil || replay.Revision != got.Revision {
		t.Fatal("accepted replay failed", e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, snap); e != nil {
		t.Fatal("composite proof restore", e)
	}
	snap.Records.Recordings[0].State = "no-timed-subtitles"
	snap.Digest, _ = snap.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("forged current recording admitted")
	}
}

func TestAdmissionStaleAndCancelledRollback(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	claim, entry := entryFixture(t, s)
	r := Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: 1, State: "no-speech", Document: json.RawMessage(`null`), SourceMap: json.RawMessage(`{}`), Provenance: json.RawMessage(`{}`), Diagnostics: json.RawMessage(`[]`)}
	current, recording, err := s.CommitAdmission(ctx, claim, 0, 0, entry, &r)
	if err != nil {
		t.Fatal(err)
	}
	nextClaim, _ := entryFixture(t, s)
	changed := current
	changed.Title = "candidate"
	before, _ := s.Revision(ctx)
	if _, _, err = s.CommitAdmission(ctx, nextClaim, 0, 0, changed, &r); err == nil {
		t.Fatal("stale recording accepted")
	}
	after, _ := s.Revision(ctx)
	actual, _ := s.Library(ctx, current.ID)
	if after != before || actual.Title != current.Title {
		t.Fatal("stale CAS partially wrote")
	}
	if _, err = s.CancelWork(ctx, contracts.ID(), nextClaim.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.CommitAdmission(ctx, nextClaim, 0, recording.Revision, changed, &r); err == nil {
		t.Fatal("cancelled work accepted")
	}
	actual, _ = s.Library(ctx, current.ID)
	if actual.Revision != current.Revision {
		t.Fatal("cancelled write changed current")
	}
}
