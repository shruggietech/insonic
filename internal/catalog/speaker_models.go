// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"math"
	"sort"
)

// SpeakerOutput is immutable output identity, independent of current corpus validity.
type SpeakerOutput struct {
	ID             string          `json:"id"`
	ModelID        string          `json:"model_id"`
	SpeakerID      string          `json:"speaker_id"`
	DatasetID      string          `json:"dataset_id"`
	WorkID         string          `json:"work_id"`
	Kind           string          `json:"kind"`
	Name           string          `json:"name"`
	Metadata       json.RawMessage `json:"metadata"`
	PublicationIDs json.RawMessage `json:"publication_ids"`
}
type SpeakerProfile struct {
	ID        string `json:"id"`
	Revision  int64  `json:"revision"`
	VersionID string `json:"version_id"`
	State     string `json:"state"`
}
type SpeakerCheckpoint struct {
	ID            string `json:"id"`
	WorkID        string `json:"work_id"`
	Attempt       int64  `json:"attempt"`
	Step          int64  `json:"step"`
	BaseDigest    string `json:"base_digest"`
	Compatibility string `json:"compatibility"`
	AdapterDigest string `json:"adapter_digest"`
	PublicationID string `json:"publication_id"`
}
type SpeakerDataset struct {
	ManifestDigest  string             `json:"manifest_digest,omitempty"`
	Dataset         Dataset            `json:"dataset"`
	SpeakerRevision int64              `json:"speaker_revision"`
	Epoch           string             `json:"epoch"`
	Recipe          json.RawMessage    `json:"recipe"`
	Summary         json.RawMessage    `json:"summary"`
	References      []CurrentReference `json:"references"`
}
type SpeakerMatchingSnapshot struct {
	Recording        Recording        `json:"recording"`
	Roster           Roster           `json:"roster"`
	Mappings         []SpeakerMapping `json:"-"`
	MappingDigest    string           `json:"mapping_digest"`
	ManualVoices     []string         `json:"manual_voices"`
	Profiles         []SpeakerProfile `json:"profiles"`
	Outputs          []SpeakerOutput  `json:"outputs"`
	Missing          []string         `json:"missing"`
	SpeakerRevisions map[string]int64 `json:"speaker_revisions"`
	Digest           string           `json:"digest"`
}
type SpeakerMatchDecision struct {
	LocalSpeakerID string          `json:"local_speaker_id"`
	SpeakerID      string          `json:"speaker_id,omitempty"`
	State          string          `json:"state"`
	Score          *float64        `json:"score,omitempty"`
	RunnerUpScore  *float64        `json:"runner_up_score,omitempty"`
	CueIDs         []string        `json:"cue_ids"`
	Diagnostics    json.RawMessage `json:"diagnostics"`
}

type speakerDatasetOptions struct {
	SpeakerRevision int64           `json:"speaker_revision"`
	Epoch           string          `json:"epoch"`
	Recipe          json.RawMessage `json:"recipe"`
	Summary         json.RawMessage `json:"summary"`
}

func SpeakerDatasetID(op, speakerID string) string {
	return operationID("speaker-dataset", op, speakerID)
}
func SpeakerDatasetSegmentID(datasetID string, ref CurrentReference) string {
	key, _ := intent(ref)
	return operationID("dataset-segment", datasetID, key)
}
func SpeakerTrainingRunID(workID string) string {
	return operationID("speaker-training-run", workID)
}
func referenceOnlySpeakerManifest(raw json.RawMessage) bool {
	if schemas.ValidateSpeakerDocument(raw) != nil || !nonsecret(raw) {
		return false
	}
	var document map[string]json.RawMessage
	if strict(raw, &document) != nil || string(document["kind"]) != `"speaker-model"` {
		return false
	}
	var declarations []map[string]json.RawMessage
	if json.Unmarshal(document["license_declarations"], &declarations) != nil {
		return false
	}
	// A declared license credit is textual schema metadata, not speaker evidence.
	// Exempt this exact scalar path only; extensions and other attribution fields
	// remain subject to the current-only evidence guard.
	for _, declaration := range declarations {
		delete(declaration, "attribution")
	}
	document["license_declarations"], _ = json.Marshal(declarations)
	clean, err := json.Marshal(document)
	return err == nil && referenceOnlyOptions(clean)
}
func validSpeakerOutput(v SpeakerOutput) bool {
	var ids []string
	if !contracts.ValidID(v.ID) || !contracts.ValidID(v.ModelID) || !contracts.ValidID(v.SpeakerID) || !contracts.ValidID(v.DatasetID) || !contracts.ValidID(v.WorkID) || !identityText(v.Kind, 128) || !identityText(v.Name, 512) || len(v.Metadata) > 1<<20 || !referenceOnlySpeakerManifest(v.Metadata) || strict(v.PublicationIDs, &ids) != nil || len(ids) > 128 {
		return false
	}
	var m struct {
		State     string `json:"state"`
		ModelID   string `json:"model_family_id"`
		ID        string `json:"model_version_id"`
		SpeakerID string `json:"originating_speaker_id"`
		Name      string `json:"display_name"`
		Training  struct {
			RunID   string `json:"training_run_id"`
			WorkID  string `json:"job_id"`
			Dataset struct {
				ID        string `json:"dataset_snapshot_id"`
				SpeakerID string `json:"originating_speaker_id"`
			} `json:"dataset"`
		} `json:"training"`
		Compatibility struct {
			Kind string `json:"model_kind"`
		} `json:"compatibility"`
	}
	if json.Unmarshal(v.Metadata, &m) != nil || m.ModelID != v.ModelID || m.ID != v.ID || m.SpeakerID != v.SpeakerID || m.Name != v.Name || m.Training.RunID != SpeakerTrainingRunID(v.WorkID) || m.Training.WorkID != v.WorkID || m.Training.Dataset.ID != v.DatasetID || m.Training.Dataset.SpeakerID != v.SpeakerID || m.Compatibility.Kind != v.Kind {
		return false
	}
	if m.State == "current" && len(ids) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !contracts.ValidID(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
func validSpeakerProfile(v SpeakerProfile) bool {
	return contracts.ValidID(v.ID) && v.Revision > 0 && (v.State == "active" && contracts.ValidID(v.VersionID) || v.State == "cleared" && v.VersionID == "")
}
func validSpeakerCheckpoint(v SpeakerCheckpoint) bool {
	return contracts.ValidID(v.ID) && contracts.ValidID(v.WorkID) && contracts.ValidID(v.PublicationID) && v.Attempt > 0 && v.Step >= 0 && (v.BaseDigest == "" || digestPattern.MatchString(v.BaseDigest)) && digestPattern.MatchString(v.AdapterDigest) && identityText(v.Compatibility, 256)
}
func validCheckpointBase(v SpeakerCheckpoint, work Work) bool {
	if work.Kind != "models.train" {
		return digestPattern.MatchString(v.BaseDigest)
	}
	var payload struct {
		Options struct {
			BaseID     string `json:"base_model_id"`
			BaseDigest string `json:"base_digest"`
		} `json:"options"`
	}
	if json.Unmarshal(work.Payload, &payload) != nil || (payload.Options.BaseID == "") != (payload.Options.BaseDigest == "") {
		return false
	}
	return v.BaseDigest == payload.Options.BaseDigest
}
func legacyMappingDigest(ms []SpeakerMapping) string {
	// Historical receipts omit the added origin fields. Only independently grounded
	// mappings can use that proof; automatic provenance can never be downgraded.
	old := []struct {
		ID             string `json:"id"`
		RecordingID    string `json:"recording_id"`
		LocalSpeakerID string `json:"local_speaker_id"`
		SpeakerID      string `json:"speaker_id"`
		DocumentDigest string `json:"document_digest"`
		Revision       int64  `json:"revision"`
	}{}
	for _, m := range ms {
		if m.Origin != "manual" || string(m.Provenance) != "{}" {
			return ""
		}
		old = append(old, struct {
			ID             string `json:"id"`
			RecordingID    string `json:"recording_id"`
			LocalSpeakerID string `json:"local_speaker_id"`
			SpeakerID      string `json:"speaker_id"`
			DocumentDigest string `json:"document_digest"`
			Revision       int64  `json:"revision"`
		}{m.ID, m.RecordingID, m.LocalSpeakerID, m.SpeakerID, m.DocumentDigest, m.Revision})
	}
	d, _ := intent(old)
	return d
}
func (s *Store) speakerDatasetTx(ctx context.Context, tx *sql.Tx, id string) (out SpeakerDataset, e error) {
	out.References = []CurrentReference{}
	if e = s.domainTx(ctx, tx, "Datasets", id, &out.Dataset); e != nil {
		return
	}
	if out.Dataset.ManifestArtifactID != nil {
		var artifact Artifact
		if e = s.domainTx(ctx, tx, "Artifacts", *out.Dataset.ManifestArtifactID, &artifact); e != nil {
			return
		}
		out.ManifestDigest = artifact.Digest
	}
	if out.Dataset.State != "current" {
		out.Recipe = json.RawMessage(`{}`)
		out.Summary = json.RawMessage(`{}`)
		return
	}
	var options speakerDatasetOptions
	if strict(out.Dataset.Options, &options) != nil {
		return out, contracts.Fail("invalid_request")
	}
	out.SpeakerRevision, out.Epoch, out.Recipe, out.Summary = options.SpeakerRevision, options.Epoch, options.Recipe, options.Summary
	records, err := s.readRecords(ctx, tx, "Members", "dataset_id", id)
	if err != nil {
		return out, err
	}
	sort.Slice(records.Members, func(i, j int) bool { return records.Members[i].Ordinal < records.Members[j].Ordinal })
	for _, m := range records.Members {
		var seg Segment
		if e = s.domainTx(ctx, tx, "Segments", m.SegmentID, &seg); e != nil {
			return
		}
		var r Recording
		if e = s.domainTx(ctx, tx, "Recordings", seg.RecordingID, &r); e != nil {
			return
		}
		var mapping SpeakerMapping
		if e = s.domainTx(ctx, tx, "SpeakerMappings", operationID("speaker-mapping", seg.RecordingID, seg.LocalSpeakerID), &mapping); e != nil {
			return
		}
		raw, _ := canonical(r.SourceMap)
		out.References = append(out.References, CurrentReference{RecordingID: r.ID, RecordingRevision: r.Revision, DocumentDigest: seg.DocumentDigest, CueID: seg.CueID, LocalSpeakerID: seg.LocalSpeakerID, SpeakerID: out.Dataset.SpeakerID, MappingRevision: mapping.Revision, SourceDigest: r.SourceDigest, SourceMapDigest: hash(raw)})
	}
	return
}
func (s *Store) SpeakerDataset(ctx context.Context, id string) (out SpeakerDataset, e error) {
	if !contracts.ValidID(id) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	out, e = s.speakerDatasetTx(ctx, tx, id)
	return out, sanitize(e)
}
func (s *Store) SpeakerDatasets(ctx context.Context, speakerID string) (out []SpeakerDataset, e error) {
	out = []SpeakerDataset{}
	if speakerID != "" && !contracts.ValidID(speakerID) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	where := ""
	var args []any
	if speakerID != "" {
		where = "speaker_id=?"
		args = []any{speakerID}
	}
	ids, e := s.identityIDs(ctx, tx, "training_dataset", where, args, "", 1001)
	if e != nil {
		return out, sanitize(e)
	}
	if len(ids) > 1000 {
		return out, contracts.Fail("output_limit")
	}
	for _, id := range ids {
		v, err := s.speakerDatasetTx(ctx, tx, id)
		if err != nil {
			return out, sanitize(err)
		}
		out = append(out, v)
	}
	return
}
func (s *Store) validateCurrentReferenceTx(ctx context.Context, tx *sql.Tx, ref CurrentReference) error {
	var r Recording
	var m SpeakerMapping
	if e := s.domainTx(ctx, tx, "Recordings", ref.RecordingID, &r); e != nil {
		return e
	}
	if e := s.domainTx(ctx, tx, "SpeakerMappings", operationID("speaker-mapping", ref.RecordingID, ref.LocalSpeakerID), &m); e != nil {
		return e
	}
	raw, e := canonical(r.SourceMap)
	if e != nil {
		return e
	}
	if r.Revision != ref.RecordingRevision || r.DocumentDigest != ref.DocumentDigest || r.SourceDigest != ref.SourceDigest || hash(raw) != ref.SourceMapDigest || m.Revision != ref.MappingRevision || m.SpeakerID != ref.SpeakerID || m.DocumentDigest != ref.DocumentDigest || m.Origin != "manual" {
		return contracts.Fail("conflict")
	}
	_, e = s.segmentCue(ctx, tx, Segment{RecordingID: ref.RecordingID, DocumentDigest: ref.DocumentDigest, CueID: ref.CueID, LocalSpeakerID: ref.LocalSpeakerID})
	return e
}
func (s *Store) CreateSpeakerDataset(ctx context.Context, op, speakerID string, refs []CurrentReference, epoch string, recipe, summary json.RawMessage, manifestPublicationID string) (out SpeakerDataset, e error) {
	if !contracts.ValidID(op) || !contracts.ValidID(speakerID) || !contracts.ValidID(manifestPublicationID) || !digestPattern.MatchString(epoch) || len(refs) > 10000 || len(recipe) > 1<<20 || len(summary) > 16384 || !referenceOnlyOptions(recipe) || !referenceOnlyOptions(summary) {
		return out, contracts.Fail("invalid_request")
	}
	id := SpeakerDatasetID(op, speakerID)
	digest, e := intent([]any{"speaker-dataset", speakerID, refs, epoch, recipe, summary, manifestPublicationID})
	if e != nil {
		return out, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, err := s.replay(ctx, tx, op, digest)
		if err != nil {
			return err
		}
		if ok {
			out, err = s.speakerDatasetTx(ctx, tx, id)
			return err
		}
		sp, err := s.speakerTx(ctx, tx, speakerID)
		if err != nil {
			return err
		}
		if sp.Speaker.State != "active" {
			return contracts.Fail("conflict")
		}
		current, err := s.selectionEpoch(ctx, tx, SpeakerSelection{SpeakerID: speakerID, ConfirmedOnly: true})
		if err != nil {
			return err
		}
		if current != epoch {
			return contracts.Fail("conflict")
		}
		pub, err := s.publicationTx(ctx, tx, manifestPublicationID)
		if err != nil {
			return err
		}
		if pub.State != "available" {
			return contracts.Fail("unavailable")
		}
		options, _ := json.Marshal(speakerDatasetOptions{sp.Speaker.Revision, epoch, recipe, summary})
		dataset := Dataset{ID: id, SpeakerID: speakerID, ManifestArtifactID: &pub.ArtifactID, Options: options, State: "current"}
		records := Records{Datasets: []Dataset{dataset}, Members: []DatasetMember{}, Segments: []Segment{}}
		seen := map[string]bool{}
		for i, ref := range refs {
			if ref.SpeakerID != speakerID {
				return contracts.Fail("invalid_request")
			}
			if err = s.validateCurrentReferenceTx(ctx, tx, ref); err != nil {
				return err
			}
			key, _ := intent(ref)
			if seen[key] {
				return contracts.Fail("invalid_request")
			}
			seen[key] = true
			sid := SpeakerDatasetSegmentID(id, ref)
			records.Segments = append(records.Segments, Segment{ID: sid, Revision: ref.RecordingRevision, RecordingID: ref.RecordingID, DocumentDigest: ref.DocumentDigest, CueID: ref.CueID, LocalSpeakerID: ref.LocalSpeakerID})
			records.Members = append(records.Members, DatasetMember{ID: operationID("dataset-member", id, key), DatasetID: id, Ordinal: int64(i), SegmentID: sid, SegmentRevision: ref.RecordingRevision})
		}
		if err = s.insertRecords(ctx, tx, records); err != nil {
			return err
		}
		proof, _ := intent(dataset)
		memberProof, _ := intent([]any{records.Members, records.Segments})
		_, err = s.accept(ctx, tx, op, digest, rev, map[string]any{"speaker_dataset_proofs": map[string]string{id: proof}, "speaker_dataset_member_proofs": map[string]string{id: memberProof}, "graph_dirty": true})
		if err != nil {
			return err
		}
		out, err = s.speakerDatasetTx(ctx, tx, id)
		return err
	})
	return out, e
}
func (s *Store) validCurrentDatasetTx(ctx context.Context, tx *sql.Tx, id string) (Dataset, error) {
	var d Dataset
	if e := s.domainTx(ctx, tx, "Datasets", id, &d); e != nil {
		return d, e
	}
	if d.State != "current" {
		return d, contracts.Fail("conflict")
	}
	var options speakerDatasetOptions
	if strict(d.Options, &options) != nil {
		return d, contracts.Fail("invalid_request")
	}
	var sp Speaker
	if e := s.domainTx(ctx, tx, "Speakers", d.SpeakerID, &sp); e != nil {
		return d, e
	}
	epoch, e := s.selectionEpoch(ctx, tx, SpeakerSelection{SpeakerID: d.SpeakerID, ConfirmedOnly: true})
	if e != nil {
		return d, e
	}
	if sp.State != "active" || sp.Revision != options.SpeakerRevision || epoch != options.Epoch {
		return d, contracts.Fail("conflict")
	}
	return d, nil
}
func (s *Store) SpeakerOutput(ctx context.Context, id string) (out SpeakerOutput, e error) {
	e = s.readOne(ctx, "SpeakerOutputs", id, &out)
	return
}
func (s *Store) SpeakerOutputs(ctx context.Context, speakerID string) (out []SpeakerOutput, e error) {
	if speakerID != "" && !contracts.ValidID(speakerID) {
		return nil, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return nil, sanitize(e)
	}
	defer tx.Rollback()
	where := ""
	var args []any
	if speakerID != "" {
		where = "speaker_id=? OR model_id IN (SELECT a.model_id FROM model_association a WHERE a.workspace_id=? AND a.speaker_id=? AND a.reason<>'profile-cleared' AND a.revision=(SELECT max(b.revision) FROM model_association b WHERE b.workspace_id=a.workspace_id AND b.model_id=a.model_id AND b.speaker_id=a.speaker_id))"
		args = []any{speakerID, s.workspace, speakerID}
	}
	ids, e := s.identityIDs(ctx, tx, "speaker_output", where, args, "", 1001)
	if e != nil {
		return nil, sanitize(e)
	}
	out = []SpeakerOutput{}
	if len(ids) > 1000 {
		return nil, contracts.Fail("output_limit")
	}
	for _, id := range ids {
		var v SpeakerOutput
		if e = s.domainTx(ctx, tx, "SpeakerOutputs", id, &v); e != nil {
			return nil, sanitize(e)
		}
		out = append(out, v)
	}
	return out, sanitize(e)
}
func (s *Store) CommitSpeakerOutput(ctx context.Context, claim Work, output SpeakerOutput) (out SpeakerOutput, e error) {
	if !validSpeakerOutput(output) || output.WorkID != claim.ID {
		return out, contracts.Fail("invalid_request")
	}
	var identity struct {
		State string `json:"state"`
	}
	var publicationIDs []string
	if json.Unmarshal(output.Metadata, &identity) != nil || identity.State != "current" || json.Unmarshal(output.PublicationIDs, &publicationIDs) != nil || len(publicationIDs) < 1 {
		return out, contracts.Fail("invalid_request")
	}
	op := operationID("speaker-output", claim.ID, output.ID)
	digest, e := intent(output)
	if e != nil {
		return out, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if _, err := s.workAuthority(ctx, tx, claim); err != nil {
			return err
		}
		_, ok, err := s.replay(ctx, tx, op, digest)
		if err != nil {
			return err
		}
		if ok {
			return s.domainTx(ctx, tx, "SpeakerOutputs", output.ID, &out)
		}
		d, err := s.validCurrentDatasetTx(ctx, tx, output.DatasetID)
		if err != nil {
			return err
		}
		if d.SpeakerID != output.SpeakerID {
			return contracts.Fail("conflict")
		}
		if err = s.availablePublications(ctx, tx, output.PublicationIDs); err != nil {
			return err
		}
		records := Records{}
		var family Model
		err = s.domainTx(ctx, tx, "Models", output.ModelID, &family)
		if err == sql.ErrNoRows {
			records.Models = []Model{{ID: output.ModelID, SpeakerID: output.SpeakerID, Name: output.Name}}
			records.ModelAssociations = []ModelAssociation{{ID: operationID("model-association", output.ModelID, output.SpeakerID), ModelID: output.ModelID, SpeakerID: output.SpeakerID, Revision: rev + 1, Reason: "elected-training"}}
		} else if err != nil {
			return err
		} else if family.SpeakerID != output.SpeakerID || family.Name != output.Name {
			return contracts.Fail("conflict")
		}
		var ids []string
		json.Unmarshal(output.PublicationIDs, &ids)
		manifest, err := s.publicationTx(ctx, tx, ids[0])
		if err != nil {
			return err
		}
		runID := SpeakerTrainingRunID(claim.ID)
		var metadata struct {
			Training struct {
				Preparation *struct {
					Artifact struct {
						ID string `json:"artifact_id"`
					} `json:"artifact"`
				} `json:"preparation"`
			} `json:"training"`
		}
		if json.Unmarshal(output.Metadata, &metadata) != nil {
			return contracts.Fail("invalid_request")
		}
		var preparationID *string
		if metadata.Training.Preparation != nil {
			artifactID := metadata.Training.Preparation.Artifact.ID
			found := false
			for _, id := range ids {
				p, err := s.publicationTx(ctx, tx, id)
				if err != nil {
					return err
				}
				if p.ArtifactID == artifactID {
					found = true
					break
				}
			}
			if !found {
				return contracts.Fail("invalid_request")
			}
			preparationID = &artifactID
		}
		records.Runs = []TrainingRun{{ID: runID, DatasetID: d.ID, SpeakerID: d.SpeakerID, JobID: claim.ID, PreparationArtifactID: preparationID, Adapter: "speaker-model", Options: json.RawMessage(`{}`), State: "current"}}
		records.Versions = []ModelVersion{{ID: output.ID, ModelID: output.ModelID, RunID: runID, DatasetID: d.ID, ManifestArtifactID: &manifest.ArtifactID, Kind: output.Kind, State: "current"}}
		for i, id := range ids {
			p, err := s.publicationTx(ctx, tx, id)
			if err != nil {
				return err
			}
			if i == 0 || preparationID != nil && p.ArtifactID == *preparationID {
				continue
			}
			records.ModelArtifacts = append(records.ModelArtifacts, ModelArtifact{ID: operationID("speaker-model-artifact", output.ID, id), VersionID: output.ID, ArtifactID: p.ArtifactID, Role: "model", Format: p.Kind})
		}
		records.SpeakerOutputs = []SpeakerOutput{output}
		if err = s.insertRecords(ctx, tx, records); err != nil {
			return err
		}
		associations := map[string]string{}
		for _, a := range records.ModelAssociations {
			associations[a.ID], _ = intent(a)
		}
		_, err = s.accept(ctx, tx, op, digest, rev, map[string]any{"speaker_output_proofs": map[string]string{output.ID: digest}, "speaker_association_proofs": associations, "speaker_model_id": output.ModelID, "graph_dirty": true})
		out = output
		return err
	})
	return
}
func (s *Store) SpeakerProfile(ctx context.Context, id string) (out SpeakerProfile, e error) {
	e = s.readOne(ctx, "SpeakerProfiles", id, &out)
	if ce, ok := e.(*contracts.Error); ok && ce.Code == "not_found" {
		out = SpeakerProfile{ID: id, State: "cleared"}
		e = nil
	}
	return
}
func (s *Store) SetSpeakerProfile(ctx context.Context, op, speakerID string, expected int64, versionID string) (out SpeakerProfile, e error) {
	if !contracts.ValidID(op) || !contracts.ValidID(speakerID) || expected < 0 || versionID != "" && !contracts.ValidID(versionID) {
		return out, contracts.Fail("invalid_request")
	}
	digest, _ := intent([]any{"speaker-profile", speakerID, expected, versionID})
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, err := s.replay(ctx, tx, op, digest)
		if err != nil {
			return err
		}
		if ok {
			return s.domainTx(ctx, tx, "SpeakerProfiles", speakerID, &out)
		}
		var sp Speaker
		if err = s.domainTx(ctx, tx, "Speakers", speakerID, &sp); err != nil {
			return err
		}
		if sp.State != "active" {
			return contracts.Fail("conflict")
		}
		var old SpeakerProfile
		err = s.domainTx(ctx, tx, "SpeakerProfiles", speakerID, &old)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if old.Revision != expected {
			return contracts.Fail("conflict")
		}
		if old.State == "active" {
			var prior SpeakerOutput
			if err = s.domainTx(ctx, tx, "SpeakerOutputs", old.VersionID, &prior); err != nil {
				return err
			}
			association := ModelAssociation{ID: operationID("profile-association", speakerID, prior.ModelID, op, "cleared"), ModelID: prior.ModelID, SpeakerID: speakerID, Revision: rev + 1, Reason: "profile-cleared"}
			if err = s.insertRecords(ctx, tx, Records{ModelAssociations: []ModelAssociation{association}}); err != nil {
				return err
			}
		}
		out = SpeakerProfile{ID: speakerID, Revision: rev + 1, VersionID: versionID, State: "cleared"}
		if versionID != "" {
			var v SpeakerOutput
			if err = s.domainTx(ctx, tx, "SpeakerOutputs", versionID, &v); err != nil {
				return err
			}
			if err = s.availablePublications(ctx, tx, v.PublicationIDs); err != nil {
				return err
			}
			// Explicit profile election supplies current association authority,
			// preserving the immutable originating speaker on the model output.
			association := ModelAssociation{ID: operationID("profile-association", speakerID, v.ModelID, op, "active"), ModelID: v.ModelID, SpeakerID: speakerID, Revision: rev + 1, Reason: "profile-election"}
			if old.State == "active" {
				var prior SpeakerOutput
				if err = s.domainTx(ctx, tx, "SpeakerOutputs", old.VersionID, &prior); err != nil {
					return err
				}
				if prior.ModelID == v.ModelID {
					if _, err = s.exec(ctx, tx, "DELETE FROM model_association WHERE workspace_id=? AND id=?", s.workspace, operationID("profile-association", speakerID, prior.ModelID, op, "cleared")); err != nil {
						return err
					}
				}
			}
			if err = s.insertRecords(ctx, tx, Records{ModelAssociations: []ModelAssociation{association}}); err != nil {
				return err
			}
			out.State = "active"
		}
		if err = s.putDomain(ctx, tx, "SpeakerProfiles", out); err != nil {
			return err
		}
		proof, _ := intent(out)
		associations := map[string]string{}
		for _, state := range []string{"active", "cleared"} {
			for _, version := range []string{old.VersionID, out.VersionID} {
				if version == "" {
					continue
				}
				var v SpeakerOutput
				if err = s.domainTx(ctx, tx, "SpeakerOutputs", version, &v); err != nil {
					return err
				}
				id := operationID("profile-association", speakerID, v.ModelID, op, state)
				var a ModelAssociation
				if err = s.domainTx(ctx, tx, "ModelAssociations", id, &a); err == sql.ErrNoRows {
					continue
				} else if err != nil {
					return err
				}
				associations[id], _ = intent(a)
			}
		}
		_, err = s.accept(ctx, tx, op, digest, rev, map[string]any{"speaker_profile_proofs": map[string]string{speakerID: proof}, "speaker_association_proofs": associations, "graph_dirty": true})
		return err
	})
	return
}
func (s *Store) SpeakerCheckpoint(ctx context.Context, id string) (out SpeakerCheckpoint, e error) {
	e = s.readOne(ctx, "SpeakerCheckpoints", id, &out)
	return
}
func (s *Store) CommitSpeakerCheckpoint(ctx context.Context, claim Work, v SpeakerCheckpoint) (out SpeakerCheckpoint, e error) {
	if !validSpeakerCheckpoint(v) || v.WorkID != claim.ID || v.Attempt != claim.Generation {
		return out, contracts.Fail("invalid_request")
	}
	op := operationID("speaker-checkpoint", claim.ID, v.ID)
	digest, _ := intent(v)
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if current, err := s.workAuthority(ctx, tx, claim); err != nil {
			return err
		} else if !validCheckpointBase(v, current) {
			return contracts.Fail("invalid_request")
		}
		_, ok, err := s.replay(ctx, tx, op, digest)
		if err != nil {
			return err
		}
		if ok {
			return s.domainTx(ctx, tx, "SpeakerCheckpoints", v.ID, &out)
		}
		if err = s.availablePublications(ctx, tx, publicationList(&v.PublicationID)); err != nil {
			return err
		}
		if err = s.insertRecords(ctx, tx, Records{SpeakerCheckpoints: []SpeakerCheckpoint{v}}); err != nil {
			return err
		}
		_, err = s.accept(ctx, tx, op, digest, rev, map[string]any{"speaker_checkpoint_proofs": map[string]string{v.ID: digest}})
		out = v
		return err
	})
	return
}
func matchingSnapshotDigest(v SpeakerMatchingSnapshot) string {
	v.Digest = ""
	d, _ := intent(v)
	return d
}
func (s *Store) freezeSpeakerMatchingTx(ctx context.Context, tx *sql.Tx, id string) (out SpeakerMatchingSnapshot, e error) {
	out.Profiles = []SpeakerProfile{}
	out.Outputs = []SpeakerOutput{}
	out.Missing = []string{}
	out.SpeakerRevisions = map[string]int64{}
	if e = s.domainTx(ctx, tx, "Recordings", id, &out.Recording); e != nil {
		return
	}
	if out.Recording.State != "ready" {
		return out, contracts.Fail("conflict")
	}
	out.Recording.Document = json.RawMessage(`null`)
	out.Recording.Provenance = json.RawMessage(`{}`)
	out.Recording.Diagnostics = json.RawMessage(`{}`)
	if out.Roster, e = s.rosterTx(ctx, tx, id); e != nil {
		return
	}
	if out.Mappings, e = s.recordingMappings(ctx, tx, id); e != nil {
		return
	}
	out.MappingDigest, _ = intent(out.Mappings)
	out.ManualVoices = []string{}
	for _, m := range out.Mappings {
		if m.Origin == "manual" {
			out.ManualVoices = append(out.ManualVoices, m.LocalSpeakerID)
		}
	}
	sort.Strings(out.ManualVoices)
	for i := range out.Roster.Members {
		out.Roster.Members[i].Name = ""
	}
	for _, sp := range out.Roster.Members {
		out.SpeakerRevisions[sp.ID] = sp.Revision
		var head SpeakerProfile
		err := s.domainTx(ctx, tx, "SpeakerProfiles", sp.ID, &head)
		if err != nil && err != sql.ErrNoRows {
			return out, err
		}
		if err == sql.ErrNoRows || head.State != "active" || sp.State != "active" {
			out.Missing = append(out.Missing, sp.ID)
			continue
		}
		var v SpeakerOutput
		if err = s.domainTx(ctx, tx, "SpeakerOutputs", head.VersionID, &v); err != nil {
			return out, err
		}
		if e = s.validateProfileAssociationTx(ctx, tx, head, v); e != nil {
			return out, e
		}
		if err = s.availablePublications(ctx, tx, v.PublicationIDs); err != nil {
			out.Missing = append(out.Missing, sp.ID)
			continue
		}
		out.Profiles = append(out.Profiles, head)
		out.Outputs = append(out.Outputs, v)
	}
	out.Digest = matchingSnapshotDigest(out)
	return
}
func (s *Store) FreezeSpeakerMatching(ctx context.Context, id string) (out SpeakerMatchingSnapshot, e error) {
	if !contracts.ValidID(id) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	out, e = s.freezeSpeakerMatchingTx(ctx, tx, id)
	return out, sanitize(e)
}
func (s *Store) CommitSpeakerMatching(ctx context.Context, claim Work, frozen SpeakerMatchingSnapshot, decisions []SpeakerMatchDecision, provenance json.RawMessage) (out []SpeakerMapping, e error) {
	out = []SpeakerMapping{}
	if !digestPattern.MatchString(frozen.Digest) || matchingSnapshotDigest(frozen) != frozen.Digest || len(decisions) > 1000 || len(provenance) > 8192 || !referenceOnlyOptions(provenance) {
		return out, contracts.Fail("invalid_request")
	}
	op := operationID("speaker-matching", claim.ID, frozen.Recording.ID)
	digest, _ := intent([]any{frozen.Digest, decisions, provenance})
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if _, err := s.workAuthority(ctx, tx, claim); err != nil {
			return err
		}
		_, ok, err := s.replay(ctx, tx, op, digest)
		if err != nil {
			return err
		}
		if ok {
			out, err = s.recordingMappings(ctx, tx, frozen.Recording.ID)
			return err
		}
		current, err := s.freezeSpeakerMatchingTx(ctx, tx, frozen.Recording.ID)
		if err != nil {
			return err
		}
		if current.Digest != frozen.Digest {
			return contracts.Fail("conflict")
		}
		var rec Recording
		if err = s.domainTx(ctx, tx, "Recordings", frozen.Recording.ID, &rec); err != nil {
			return err
		}
		voices := documentSpeakers(rec.Document)
		eligible := map[string]SpeakerOutput{}
		for _, p := range frozen.Profiles {
			for _, v := range frozen.Outputs {
				if v.ID == p.VersionID {
					eligible[p.ID] = v
				}
			}
		}
		manual := map[string]bool{}
		for _, m := range current.Mappings {
			if m.Origin == "manual" {
				manual[m.LocalSpeakerID] = true
			}
		}
		seen := map[string]bool{}
		for _, d := range decisions {
			if !contracts.ValidLocalSpeakerID(d.LocalSpeakerID) || !voices[d.LocalSpeakerID] || seen[d.LocalSpeakerID] || len(d.CueIDs) > 10000 || len(d.Diagnostics) > 8192 || !referenceOnlyOptions(d.Diagnostics) {
				return contracts.Fail("invalid_request")
			}
			seen[d.LocalSpeakerID] = true
			for _, n := range []*float64{d.Score, d.RunnerUpScore} {
				if n != nil && (math.IsNaN(*n) || math.IsInf(*n, 0)) {
					return contracts.Fail("invalid_request")
				}
			}
			switch d.State {
			case "accepted":
				if d.Score == nil || len(d.CueIDs) == 0 {
					return contracts.Fail("invalid_request")
				}
				if _, ok := eligible[d.SpeakerID]; !ok {
					return contracts.Fail("invalid_request")
				}
			case "unknown", "ambiguous", "manual-preserved":
				if d.SpeakerID != "" {
					return contracts.Fail("invalid_request")
				}
			default:
				return contracts.Fail("invalid_request")
			}
			for _, cue := range d.CueIDs {
				if _, err = s.segmentCue(ctx, tx, Segment{RecordingID: rec.ID, DocumentDigest: rec.DocumentDigest, CueID: cue, LocalSpeakerID: d.LocalSpeakerID}); err != nil {
					return err
				}
			}
		}
		if len(seen) != len(voices) {
			return contracts.Fail("invalid_request")
		}
		if _, err = s.exec(ctx, tx, "DELETE FROM speaker_mapping WHERE workspace_id=? AND recording_id=? AND origin='automatic'", s.workspace, rec.ID); err != nil {
			return err
		}
		for _, d := range decisions {
			if manual[d.LocalSpeakerID] || d.State != "accepted" {
				continue
			}
			v := eligible[d.SpeakerID]
			bounded := d
			if len(bounded.CueIDs) > 32 {
				bounded.CueIDs = bounded.CueIDs[:32]
			}
			raw, _ := json.Marshal(map[string]any{"work_id": claim.ID, "model_version_id": v.ID, "snapshot_digest": frozen.Digest, "decision": bounded, "evidence_count": len(d.CueIDs), "settings": provenance})
			m := SpeakerMapping{ID: operationID("speaker-mapping", rec.ID, d.LocalSpeakerID), RecordingID: rec.ID, LocalSpeakerID: d.LocalSpeakerID, SpeakerID: d.SpeakerID, DocumentDigest: rec.DocumentDigest, Revision: rev + 1, Origin: "automatic", Provenance: raw}
			if !validSpeakerMapping(m) {
				return contracts.Fail("invalid_request")
			}
			if err = s.putDomain(ctx, tx, "SpeakerMappings", m); err != nil {
				return err
			}
		}
		// Automatic predictions are excluded from independent training membership,
		// so this operation changes no current corpus preparation authority.
		out, err = s.recordingMappings(ctx, tx, rec.ID)
		if err != nil {
			return err
		}
		md, _ := intent(out)
		result, err := s.currentReceiptResult(ctx, tx, "mapping_recording_id", rec.ID, rev+1, recordingDigest(rec))
		if err != nil {
			return err
		}
		result["mapping_digest"] = md
		result["matching_snapshot_digest"] = frozen.Digest
		result["cleanup_entry_id"] = rec.ID
		_, err = s.accept(ctx, tx, op, digest, rev, result)
		return err
	})
	return
}
