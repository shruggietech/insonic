// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
)

// Completed output identity and binary artifacts survive correction; mutable
// current input lineage cannot retain obsolete assignment-bearing artifacts.
func (s *Store) reconcileSpeakerOutputLineage(ctx context.Context, tx *sql.Tx, result map[string]json.RawMessage) error {
	where := "dataset_id IN (SELECT id FROM training_dataset WHERE workspace_id=? AND state='invalidated') AND " + s.evidenceJSONText("metadata", "state") + "='current'"
	if s.backend == "postgresql" {
		where = "dataset_id IN (SELECT id FROM training_dataset WHERE workspace_id=? AND state='invalidated') AND " + s.evidenceJSONText("metadata::jsonb", "state") + "='current'"
	}
	ids, e := s.identityIDs(ctx, tx, "speaker_output", where, []any{s.workspace}, "", 10001)
	if e != nil {
		return e
	}
	if len(ids) > 10000 {
		return contracts.Fail("output_limit")
	}
	proofs := map[string]string{}
	for _, id := range ids {
		var output SpeakerOutput
		if e = s.domainTx(ctx, tx, "SpeakerOutputs", id, &output); e != nil {
			return e
		}
		var m map[string]json.RawMessage
		if e = strict(output.Metadata, &m); e != nil {
			return e
		}
		var training map[string]json.RawMessage
		if e = strict(m["training"], &training); e != nil {
			return e
		}
		var dataset map[string]json.RawMessage
		if e = strict(training["dataset"], &dataset); e != nil {
			return e
		}
		m["state"] = json.RawMessage(`"invalidated"`)
		m["manifest_sha256"] = json.RawMessage(`null`)
		if _, ok := m["manifest_artifact"]; ok {
			m["manifest_artifact"] = json.RawMessage(`null`)
		}
		training["preparation"] = json.RawMessage(`null`)
		dataset["state"] = json.RawMessage(`"invalidated"`)
		dataset["manifest_sha256"] = json.RawMessage(`null`)
		training["dataset"], _ = json.Marshal(dataset)
		m["training"], _ = json.Marshal(training)
		output.Metadata, _ = json.Marshal(m)
		var publications []string
		json.Unmarshal(output.PublicationIDs, &publications)
		retained := []string{}
		for _, pubID := range publications {
			p, err := s.publicationTx(ctx, tx, pubID)
			if err != nil {
				return err
			}
			var exists bool
			if e = s.row(ctx, tx, "SELECT EXISTS(SELECT 1 FROM model_artifact WHERE workspace_id=? AND version_id=? AND artifact_id=? AND role IN ('weights','checkpoint','model'))", s.workspace, output.ID, p.ArtifactID).Scan(&exists); e != nil {
				return e
			}
			if exists {
				retained = append(retained, pubID)
			}
		}
		output.PublicationIDs, _ = json.Marshal(retained)
		if !validSpeakerOutput(output) {
			return contracts.Fail("invalid_request")
		}
		if e = s.putDomain(ctx, tx, "SpeakerOutputs", output); e != nil {
			return e
		}
		proofs[id], _ = intent(output)
	}
	if len(proofs) > 0 {
		result["speaker_output_proofs"], _ = json.Marshal(proofs)
	}
	return nil
}
