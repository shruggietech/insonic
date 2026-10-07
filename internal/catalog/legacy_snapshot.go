// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strconv"
)

type snapshotEnvelope struct {
	Kind          string          `json:"kind"`
	Version       string          `json:"schema_version"`
	CatalogSchema int             `json:"catalog_schema"`
	WorkspaceID   string          `json:"workspace_id"`
	Revision      int64           `json:"revision"`
	Records       json.RawMessage `json:"records"`
	State         []TableData     `json:"state"`
	Digest        string          `json:"digest"`
}

type legacySegment struct {
	ID             string          `json:"id"`
	Revision       int64           `json:"revision"`
	AssetID        string          `json:"asset_id"`
	SpeakerID      *string         `json:"speaker_id"`
	StartUS        int64           `json:"start_us"`
	EndUS          int64           `json:"end_us"`
	Channel        int64           `json:"channel"`
	Attribution    json.RawMessage `json:"attribution"`
	ClipArtifactID *string         `json:"clip_artifact_id"`
}

func decodedSnapshot(envelope snapshotEnvelope) (Snapshot, error) {
	out := Snapshot{Kind: envelope.Kind, Version: envelope.Version, CatalogSchema: envelope.CatalogSchema, WorkspaceID: envelope.WorkspaceID, Revision: envelope.Revision, State: envelope.State, Digest: envelope.Digest}
	if envelope.CatalogSchema < 1 || envelope.CatalogSchema > 3 {
		return out, strict(envelope.Records, &out.Records)
	}
	// Verify the original legacy representation before deleting its copied
	// assignments. Canonicalization retains exact integer tokens and raw strings.
	want := envelope.Digest
	envelope.Digest = ""
	digest, e := intent(envelope)
	if e != nil || digest != want {
		return out, contracts.Fail("invalid_request")
	}
	var fields map[string]json.RawMessage
	if strict(envelope.Records, &fields) != nil {
		return out, contracts.Fail("invalid_request")
	}
	var segments []legacySegment
	if raw, ok := fields["segments"]; ok && strict(raw, &segments) != nil {
		return out, contracts.Fail("invalid_request")
	}
	for _, segment := range segments {
		if !contracts.ValidID(segment.ID) || !contracts.ValidID(segment.AssetID) || segment.Revision < 1 || segment.StartUS < 0 || segment.EndUS <= segment.StartUS || segment.Channel < 0 || !validJSON(segment.Attribution) {
			return out, contracts.Fail("invalid_request")
		}
		if segment.ClipArtifactID != nil {
			out.legacyDerivedArtifacts = append(out.legacyDerivedArtifacts, *segment.ClipArtifactID)
		}
	}
	var members []DatasetMember
	if raw, ok := fields["members"]; ok && strict(raw, &members) != nil {
		return out, contracts.Fail("invalid_request")
	}
	segmentKeys := map[string]bool{}
	for _, segment := range segments {
		segmentKeys[segment.ID+":"+strconv.FormatInt(segment.Revision, 10)] = true
	}
	for _, member := range members {
		if validateRecord(member) != nil || !segmentKeys[member.SegmentID+":"+strconv.FormatInt(member.SegmentRevision, 10)] {
			return out, contracts.Fail("invalid_request")
		}
	}
	fields["segments"] = json.RawMessage("null")
	fields["members"] = json.RawMessage("null")
	raw, e := json.Marshal(fields)
	if e != nil || strict(raw, &out.Records) != nil {
		return out, contracts.Fail("invalid_request")
	}
	for i := range out.Records.Datasets {
		d := &out.Records.Datasets[i]
		if d.ManifestArtifactID != nil {
			out.legacyDerivedArtifacts = append(out.legacyDerivedArtifacts, *d.ManifestArtifactID)
		}
		d.ManifestArtifactID = nil
		d.Options = json.RawMessage(`{}`)
		d.State = "invalidated"
		d.InvalidatedRevision = envelope.Revision + 1
		d.InvalidatedBy = &out.WorkspaceID
		out.legacyInvalidatedDatasets = append(out.legacyInvalidatedDatasets, d.ID)
	}
	for i := range out.Records.Runs {
		r := &out.Records.Runs[i]
		if r.PreparationArtifactID != nil {
			out.legacyDerivedArtifacts = append(out.legacyDerivedArtifacts, *r.PreparationArtifactID)
		}
		r.PreparationArtifactID = nil
		r.Options = json.RawMessage(`{}`)
		r.State = "invalidated"
	}
	for i := range out.Records.Versions {
		v := &out.Records.Versions[i]
		if v.ManifestArtifactID != nil {
			out.legacyDerivedArtifacts = append(out.legacyDerivedArtifacts, *v.ManifestArtifactID)
		}
		v.ManifestArtifactID = nil
		v.State = "invalidated"
	}
	if envelope.CatalogSchema == 1 {
		if len(out.State) != len(stateTables)-1 {
			return out, contracts.Fail("invalid_request")
		}
		out.State = append(out.State, TableData{Name: "artifact_publication", Rows: [][]json.RawMessage{}})
	}
	out.CatalogSchema = SchemaVersion
	out.Digest, e = out.digest()
	return out, e
}
