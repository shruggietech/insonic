package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"time"
)

func TestSQLiteCorpusAssociations(t *testing.T) { corpusSuite(t, localStore(t, contracts.ID())) }
func corpusSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	job, e := s.StartJob(ctx, contracts.ID(), contracts.ID(), 10, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	source, manifest, weights := contracts.ID(), contracts.ID(), contracts.ID()
	asset, speaker, segment, dataset, run, model, version := contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID()
	rows := Records{
		Artifacts: []Artifact{{source, "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", 1, "source"}, {manifest, "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", 1, "manifest"}, {weights, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", 1, "model"}},
		Assets:    []Asset{{asset, source, "original", 1000000}}, Speakers: []Speaker{{speaker, "Speaker"}},
		Segments: []Segment{{ID: segment, Revision: 1, AssetID: asset, SpeakerID: &speaker, StartUS: 0, EndUS: 1000000, Channel: 0, Attribution: json.RawMessage(`{"basis":"selected"}`)}},
		Datasets: []Dataset{{dataset, speaker, manifest, json.RawMessage(`{"recipe_version":"1"}`)}}, Members: []DatasetMember{{contracts.ID(), dataset, 0, segment, 1}},
		Runs: []TrainingRun{{run, dataset, speaker, job.JobID, manifest, "fixture", json.RawMessage(`{}`)}}, Models: []Model{{model, speaker, "Family"}}, Versions: []ModelVersion{{version, model, run, dataset, manifest, "trained"}}, ModelArtifacts: []ModelArtifact{{contracts.ID(), version, weights, "weights", "fixture"}}, ModelAssociations: []ModelAssociation{{contracts.ID(), model, speaker, 1, "original"}},
	}
	rev, _ := s.Revision(ctx)
	if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: rows}); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(snap.Records.Members) != 1 || snap.Records.ModelArtifacts[0].ArtifactID != weights || snap.Records.Runs[0].JobID != job.JobID {
		t.Fatal("typed provenance lost")
	}
	rev, _ = s.Revision(ctx)
	if _, e = s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: Records{Members: []DatasetMember{{contracts.ID(), dataset, 1, segment, 1}}}}); e == nil {
		t.Fatal("frozen dataset mutated")
	}
	empty := localStore(t, s.workspace)
	if e = empty.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	restored, e := empty.Export(ctx)
	if e != nil || len(restored.Records.ModelArtifacts) != 1 || restored.Records.Members[0].SegmentRevision != 1 {
		t.Fatal("reserved associations failed round trip", e)
	}
}
