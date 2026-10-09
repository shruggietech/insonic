// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"sort"
)

func (s *Store) validateSpeakerModelState(ctx context.Context, tx *sql.Tx, r Records) error {
	expected := map[string]map[string]string{"speaker_output_proofs": {}, "speaker_profile_proofs": {}, "speaker_checkpoint_proofs": {}, "speaker_dataset_proofs": {}, "speaker_association_proofs": {}, "speaker_dataset_member_proofs": {}}
	rows, e := tx.QueryContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			break
		}
		var result map[string]json.RawMessage
		if e = strict([]byte(raw), &result); e != nil {
			break
		}
		for k, proofs := range expected {
			if p, ok := result[k]; ok {
				var m map[string]string
				if e = strict(p, &m); e != nil {
					break
				}
				for id, d := range m {
					if !contracts.ValidID(id) || !digestPattern.MatchString(d) {
						e = contracts.Fail("invalid_request")
						break
					}
					proofs[id] = d
				}
			}
		}
	}
	re := rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if re != nil {
		return re
	}
	verify := func(key, id string, v any) error {
		d, err := intent(v)
		if err != nil || expected[key][id] != d {
			return contracts.Fail("invalid_request")
		}
		delete(expected[key], id)
		return nil
	}
	for _, v := range r.SpeakerOutputs {
		if !validSpeakerOutput(v) {
			return contracts.Fail("invalid_request")
		}
		if e = verify("speaker_output_proofs", v.ID, v); e != nil {
			return e
		}
		var version ModelVersion
		var run TrainingRun
		if e = s.domainTx(ctx, tx, "Versions", v.ID, &version); e != nil {
			return e
		}
		if e = s.domainTx(ctx, tx, "Runs", version.RunID, &run); e != nil {
			return e
		}
		if version.ModelID != v.ModelID || version.DatasetID != v.DatasetID || version.Kind != v.Kind || run.JobID != v.WorkID || run.SpeakerID != v.SpeakerID {
			return contracts.Fail("invalid_request")
		}
		var dataset Dataset
		if e = s.domainTx(ctx, tx, "Datasets", v.DatasetID, &dataset); e != nil {
			return e
		}
		var metadata struct {
			State string `json:"state"`
		}
		if json.Unmarshal(v.Metadata, &metadata) != nil || metadata.State != version.State || version.State != dataset.State || run.State != dataset.State {
			return contracts.Fail("invalid_request")
		}
		var ids []string
		json.Unmarshal(v.PublicationIDs, &ids)
		for _, id := range ids {
			p, err := s.publicationTx(ctx, tx, id)
			if err != nil {
				return err
			}
			if version.ManifestArtifactID != nil && p.ArtifactID == *version.ManifestArtifactID || run.PreparationArtifactID != nil && p.ArtifactID == *run.PreparationArtifactID {
				continue
			}
			var exists bool
			if e = s.row(ctx, tx, "SELECT EXISTS(SELECT 1 FROM model_artifact WHERE workspace_id=? AND version_id=? AND artifact_id=? AND role='model')", s.workspace, v.ID, p.ArtifactID).Scan(&exists); e != nil {
				return e
			}
			if !exists {
				return contracts.Fail("invalid_request")
			}
		}
	}
	for _, v := range r.SpeakerProfiles {
		if !validSpeakerProfile(v) {
			return contracts.Fail("invalid_request")
		}
		if e = verify("speaker_profile_proofs", v.ID, v); e != nil {
			return e
		}
		if v.State == "active" {
			var out SpeakerOutput
			if e = s.domainTx(ctx, tx, "SpeakerOutputs", v.VersionID, &out); e != nil {
				return e
			}
			if e = s.validateProfileAssociationTx(ctx, tx, v, out); e != nil {
				return e
			}
		}
	}
	for _, v := range r.SpeakerCheckpoints {
		if !validSpeakerCheckpoint(v) {
			return contracts.Fail("invalid_request")
		}
		if e = verify("speaker_checkpoint_proofs", v.ID, v); e != nil {
			return e
		}
		var work Work
		if e = s.domainTx(ctx, tx, "Works", v.WorkID, &work); e != nil {
			return e
		}
		if !validCheckpointBase(v, work) {
			return contracts.Fail("invalid_request")
		}
		if _, e = s.publicationTx(ctx, tx, v.PublicationID); e != nil {
			return e
		}
	}
	for _, v := range r.Datasets {
		if _, ok := expected["speaker_dataset_proofs"][v.ID]; !ok {
			continue
		}
		if v.State == "current" {
			if e = verify("speaker_dataset_proofs", v.ID, v); e != nil {
				return e
			}
			r, err := s.readRecords(ctx, tx, "Members", "dataset_id", v.ID)
			if err != nil {
				return err
			}
			sort.Slice(r.Members, func(i, j int) bool { return r.Members[i].Ordinal < r.Members[j].Ordinal })
			segments := []Segment{}
			for _, m := range r.Members {
				var seg Segment
				if e = s.domainTx(ctx, tx, "Segments", m.SegmentID, &seg); e != nil {
					return e
				}
				segments = append(segments, seg)
			}
			if r.Members == nil {
				r.Members = []DatasetMember{}
			}
			if e = verify("speaker_dataset_member_proofs", v.ID, []any{r.Members, segments}); e != nil {
				return e
			}
		} else {
			delete(expected["speaker_dataset_proofs"], v.ID)
			delete(expected["speaker_dataset_member_proofs"], v.ID)
		}
	}
	for _, a := range r.ModelAssociations {
		if _, ok := expected["speaker_association_proofs"][a.ID]; ok {
			if e = verify("speaker_association_proofs", a.ID, a); e != nil {
				return e
			}
		} else if a.Reason == "profile-election" || a.Reason == "profile-cleared" || a.Reason == "elected-training" {
			return contracts.Fail("invalid_request")
		}
	}
	for _, proofs := range expected {
		if len(proofs) != 0 {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
func (s *Store) validateProfileAssociationTx(ctx context.Context, tx *sql.Tx, head SpeakerProfile, output SpeakerOutput) error {
	var reason string
	e := s.row(ctx, tx, "SELECT reason FROM model_association WHERE workspace_id=? AND model_id=? AND speaker_id=? ORDER BY revision DESC LIMIT 1", s.workspace, output.ModelID, head.ID).Scan(&reason)
	if e != nil {
		return e
	}
	if reason == "profile-cleared" {
		return contracts.Fail("invalid_request")
	}
	return nil
}
