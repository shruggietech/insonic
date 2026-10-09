// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/schemas"
)

func TestPortableOutputRecordsExactArtifactsWithoutPreparedContents(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	digest := strings.Repeat("a", 64)
	opts := TrainOptions{DatasetID: id, Name: "Fixture speaker", Kind: "voice-embedding", Adapter: Adapter{ID: "pyannote-profile", ContractVersion: "1", Mode: "local", Architecture: "wespeaker", OutputKinds: []string{"voice-embedding"}, Consumers: []string{"pyannote-profile"}}}
	instant := portableInstant(time.Now())
	output := Output{StartedAt: &instant, CompletedAt: &instant, ContractVersion: "1", Kind: "voice-embedding", Architecture: "wespeaker", Consumers: []string{"pyannote-profile"}, SupportedOperations: []string{"voice-matching"}, Artifacts: []Artifact{{Role: "profile", Format: "insonic-embedding-v1", SHA256: digest, Size: 64}}}
	pub := catalog.Publication{ID: id, ArtifactID: id, Digest: digest, Size: 64, State: "available"}
	raw, e := BuildModelManifest(id, catalog.Work{ID: id, Generation: 1}, catalog.SpeakerDataset{Dataset: catalog.Dataset{ID: id, SpeakerID: id}, SpeakerRevision: 1, Epoch: digest, ManifestDigest: digest}, opts, output, []catalog.Publication{pub})
	if e != nil {
		t.Fatal(e)
	}
	if e = schemas.ValidateSpeakerDocument(raw); e != nil {
		t.Fatal(e)
	}
	var manifest struct {
		Training struct {
			RunID       string          `json:"training_run_id"`
			Preparation json.RawMessage `json:"preparation"`
		} `json:"training"`
	}
	if json.Unmarshal(raw, &manifest) != nil || manifest.Training.RunID != catalog.SpeakerTrainingRunID(id) {
		t.Fatal("portable run differs from catalog authority")
	}
	output.PreparationPublication = &pub
	prepared, e := BuildModelManifest(id, catalog.Work{ID: id, Generation: 1}, catalog.SpeakerDataset{Dataset: catalog.Dataset{ID: id, SpeakerID: id}, SpeakerRevision: 1, Epoch: digest, ManifestDigest: digest, Recipe: json.RawMessage(`{}`)}, opts, output, []catalog.Publication{pub})
	if e != nil || json.Unmarshal(prepared, &manifest) != nil || string(manifest.Training.Preparation) == "null" {
		t.Fatal("missing transient preparation proof", e)
	}
	if !strings.Contains(string(manifest.Training.Preparation), `"input_artifact_ids":[]`) {
		t.Fatal("invented retained input artifact")
	}
	files, e := ManifestArtifactDescriptors(raw)
	if e != nil || len(files) != 1 || files[0].PublicationID != id || files[0].Digest != digest {
		t.Fatalf("portable artifact %+v %v", files, e)
	}
	for _, forbidden := range []string{"speaker_attributions", "source_path", "output_directory", "vector", "transcript"} {
		if strings.Contains(string(raw), `"`+forbidden+`"`) {
			t.Fatal("prepared or assignment content leaked", forbidden)
		}
	}
	pub.Digest = strings.Repeat("b", 64)
	if _, e = BuildModelManifest(id, catalog.Work{ID: id, Generation: 1}, catalog.SpeakerDataset{Dataset: catalog.Dataset{ID: id, SpeakerID: id}, SpeakerRevision: 1, Epoch: digest, ManifestDigest: digest}, opts, output, []catalog.Publication{pub}); e == nil {
		t.Fatal("mismatched publication accepted")
	}
}

func TestPortableDatasetContainsReferencesWithoutAssignmentCopies(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	digest := strings.Repeat("a", 64)
	raw, e := BuildDatasetManifest(id, id, catalog.Speaker{ID: id, Revision: 1}, 1, Recipe{}, []catalog.CurrentReference{{RecordingID: id, RecordingRevision: 1, DocumentDigest: digest, CueID: "cue-1", LocalSpeakerID: "voice-1", SpeakerID: id, SourceDigest: digest}}, Summary{References: 1, Included: 1, DurationUS: 1000000})
	if e != nil {
		t.Fatal(e)
	}
	if e = schemas.ValidateSpeakerDocument(raw); e != nil {
		t.Fatal(e)
	}
	var document map[string]json.RawMessage
	if json.Unmarshal(raw, &document) != nil {
		t.Fatal("invalid JSON")
	}
	if !strings.Contains(string(document["members"]), `"cue_id":"cue-1"`) {
		t.Fatal("missing current reference")
	}
	for _, forbidden := range []string{"speaker_attributions", "text", "start_us", "end_us"} {
		if strings.Contains(string(raw), `"`+forbidden+`"`) {
			t.Fatal("copied evidence", forbidden)
		}
	}
}
