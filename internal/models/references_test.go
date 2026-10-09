// SPDX-License-Identifier: Apache-2.0
package models

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func TestReferenceSyntax(t *testing.T) {
	for _, v := range []string{"quick", "base:00000000-0000-4000-8000-000000000001", "speaker:00000000-0000-4000-8000-000000000001", "source:custom/org/model:latest"} {
		if !ValidateReference(v) {
			t.Fatal(v)
		}
	}
	for _, v := range []string{"", "UPPER", "a b", "unknown:a", "source:custom/", "source:custom/x\n"} {
		if ValidateReference(v) {
			t.Fatal(v)
		}
	}
	m := fixtureManifest()
	if InstallationID(m) != InstallationID(m) {
		t.Fatal("identity")
	}
	m2 := m
	m2.Revision = "new"
	if AcquisitionID("workspace", m) == AcquisitionID("workspace", m2) {
		t.Fatal("manifest omitted from acquisition identity")
	}
	frozen, e := ResolveManifest(m, "")
	if e != nil {
		t.Fatal(e)
	}
	m.Files[0].Role = "changed"
	if frozen.Manifest.Files[0].Role == "changed" {
		t.Fatal("selection shares mutable manifest")
	}
}

type lineageFixture struct {
	catalog.Catalog
	lineage catalog.SpeakerModelLineage
}

func (f lineageFixture) SpeakerModelVersion(ctx context.Context, id string) (catalog.SpeakerModelLineage, error) {
	return f.lineage, nil
}
func (f lineageFixture) Artifact(ctx context.Context, id string) (catalog.Artifact, error) {
	return catalog.Artifact{ID: id, Digest: strings.Repeat("a", 64), Size: 1, Kind: "manifest"}, nil
}
func TestTrainedAndHostedResolutionPreservesDistinctIdentity(t *testing.T) {
	version, family, speaker, run, dataset, manifest := contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID()
	fixture := lineageFixture{lineage: catalog.SpeakerModelLineage{Version: catalog.ModelVersion{ID: version, ModelID: family, RunID: run, DatasetID: dataset, ManifestArtifactID: &manifest, Kind: "embedding", State: "current"}, Model: catalog.Model{ID: family, SpeakerID: speaker, Name: "Profile"}, Run: catalog.TrainingRun{ID: run, SpeakerID: speaker, DatasetID: dataset}, Dataset: catalog.Dataset{ID: dataset, SpeakerID: speaker}, Artifacts: []catalog.ModelArtifact{{ID: contracts.ID(), VersionID: version, ArtifactID: contracts.ID(), Role: "embedding", Format: "profile"}}, Associations: []catalog.ModelAssociation{{ID: contracts.ID(), ModelID: family, SpeakerID: speaker, Revision: 1, Reason: "origin"}}}}
	got, e := NewResolver(fixture, nil).Resolve(context.Background(), "speaker:"+version, "voice-matching")
	if e != nil || got.Target.Kind != "speaker" || got.Lineage.Model.SpeakerID != speaker || got.Lineage.Run.ID != run || got.Digest != strings.Repeat("a", 64) || got.Compatible || got.Manifest != nil {
		t.Fatal("trained identity fabricated", got, e)
	}
	fixture.lineage.Version.State = "invalidated"
	fixture.lineage.Version.ManifestArtifactID = nil
	got, e = NewResolver(fixture, nil).Resolve(context.Background(), "speaker:"+version, "voice-matching")
	if e != nil || got.State != "invalidated" || got.Digest != "" {
		t.Fatal("invalidated manifest fabricated", e)
	}
	hosted := catalog.ModelTarget{Kind: "hosted", Operation: "transcription", Adapter: "insonic-http", ContractVersion: "1", Endpoint: "https://example.org/infer", RemoteModel: "handle-1", UpstreamRevision: "provider-r1"}
	got, e = NewResolver(fixture, nil).ResolveTarget(context.Background(), hosted, "transcription")
	if e != nil || got.State != "hosted-only" || !got.Compatible || got.Manifest != nil || got.Digest == "" || got.Target.RemoteModel != "handle-1" {
		t.Fatal("hosted weights fabricated", got, e)
	}
	encoded, _ := json.Marshal(got)
	if strings.Contains(string(encoded), "password") {
		t.Fatal("credential leak")
	}
}
