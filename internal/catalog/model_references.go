// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"net"
	"regexp"
)

type ModelTarget struct {
	Kind             string `json:"kind"`
	ID               string `json:"id,omitempty"`
	Operation        string `json:"operation"`
	Adapter          string `json:"adapter,omitempty"`
	ContractVersion  string `json:"contract_version,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	RemoteModel      string `json:"remote_model,omitempty"`
	CredentialID     string `json:"credential_id,omitempty"`
	UpstreamRevision string `json:"upstream_revision,omitempty"`
}
type ModelAlias struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Revision int64           `json:"revision"`
	State    string          `json:"state"`
	Target   json.RawMessage `json:"target"`
}
type ModelSource struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Revision     int64  `json:"revision"`
	State        string `json:"state"`
	URL          string `json:"url"`
	CredentialID string `json:"credential_id,omitempty"`
	LocalHTTP    bool   `json:"local_http"`
}

// Booleans use portable SQL text and decode through the typed record boundary.
var modelName = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)

func ValidModelName(v string) bool { return modelName.MatchString(v) && !contracts.ValidID(v) }
func ModelReferenceID(workspace, kind, name string) string {
	return operationID("model-reference", workspace, kind, name)
}
func ModelOperation(v string) bool {
	return v == "transcription" || v == "diarization" || v == "voice-matching" || v == "speaker-model-training"
}
func DecodeModelTarget(raw json.RawMessage) (ModelTarget, error) {
	var out ModelTarget
	if strict(raw, &out) != nil || !ModelOperation(out.Operation) {
		return out, contracts.Fail("invalid_request")
	}
	if out.Kind == "base" || out.Kind == "speaker" {
		if !contracts.ValidID(out.ID) || out.Endpoint != "" || out.RemoteModel != "" || out.CredentialID != "" || out.UpstreamRevision != "" {
			return out, contracts.Fail("invalid_request")
		}
	} else if out.Kind == "hosted" {
		if out.ID != "" || out.Adapter != "insonic-http" || out.ContractVersion != "1" || !identityText(out.RemoteModel, 256) || !identityText(out.UpstreamRevision, 512) || out.CredentialID != "" && !contracts.ValidID(out.CredentialID) {
			return out, contracts.Fail("invalid_request")
		}
		u, e := contracts.SourceURL(out.Endpoint)
		if e != nil || u.RawQuery != "" || u.ForceQuery {
			return out, contracts.Fail("invalid_request")
		}
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "https" && (u.Scheme != "http" || !(u.Hostname() == "localhost" || ip != nil && ip.IsLoopback())) {
			return out, contracts.Fail("invalid_request")
		}
	} else {
		return out, contracts.Fail("invalid_request")
	}
	if out.Adapter != "" && !identityText(out.Adapter, 128) || out.ContractVersion != "" && !identityText(out.ContractVersion, 64) || (out.Adapter == "") != (out.ContractVersion == "") {
		return out, contracts.Fail("invalid_request")
	}
	return out, nil
}
func validModelAlias(v ModelAlias) bool {
	_, e := DecodeModelTarget(v.Target)
	return contracts.ValidID(v.ID) && ValidModelName(v.Name) && (v.State == "active" || v.State == "deleted") && e == nil && nonsecret(v.Target)
}
func validModelSource(v ModelSource) bool {
	u, e := contracts.SourceURL(v.URL)
	if e != nil {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	transport := u.Scheme == "https" || v.LocalHTTP && u.Scheme == "http" && (u.Hostname() == "localhost" || ip != nil && ip.IsLoopback())
	return contracts.ValidID(v.ID) && ValidModelName(v.Name) && (v.State == "active" || v.State == "deleted") && transport && (v.CredentialID == "" || contracts.ValidID(v.CredentialID))
}
func (s *Store) modelRefRead(ctx context.Context, field, ref string, out any) error {
	if contracts.ValidID(ref) {
		return s.readOne(ctx, field, ref, out)
	}
	if !ValidModelName(ref) {
		return contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return sanitize(e)
	}
	defer tx.Rollback()
	var id string
	e = s.row(ctx, tx, "SELECT id FROM "+domainNamed(field).name+" WHERE workspace_id=? AND name=?", s.workspace, ref).Scan(&id)
	if e == nil {
		e = s.domainTx(ctx, tx, field, id, out)
	}
	return sanitize(e)
}
func (s *Store) ModelAlias(ctx context.Context, ref string) (out ModelAlias, e error) {
	e = s.modelRefRead(ctx, "ModelAliases", ref, &out)
	return
}
func (s *Store) ModelSource(ctx context.Context, ref string) (out ModelSource, e error) {
	e = s.modelRefRead(ctx, "ModelSources", ref, &out)
	return
}
func (s *Store) ModelAliases(ctx context.Context) ([]ModelAlias, error) {
	r, e := s.currentRecords(ctx, "ModelAliases")
	return r.ModelAliases, e
}
func (s *Store) ModelSources(ctx context.Context) ([]ModelSource, error) {
	r, e := s.currentRecords(ctx, "ModelSources")
	return r.ModelSources, e
}
func (s *Store) modelRefWrite(ctx context.Context, op string, expected int64, field, id, name string, value any, out any) error {
	if !contracts.ValidID(op) || expected < 0 {
		return contracts.Fail("invalid_request")
	}
	digest, e := intent([]any{field, expected, value})
	if e != nil {
		return e
	}
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, field, id, out)
		}
		var oldID string
		var oldRev int64
		var oldName string
		e = s.row(ctx, tx, "SELECT id,name,revision FROM "+domainNamed(field).name+" WHERE workspace_id=? AND (id=? OR name=?)", s.workspace, id, name).Scan(&oldID, &oldName, &oldRev)
		if e != nil && e != sql.ErrNoRows {
			return e
		}
		if e == nil && (oldID != id || oldName != name || oldRev != expected) || e == sql.ErrNoRows && expected != 0 {
			return contracts.Fail("conflict")
		}
		if v, ok := value.(ModelAlias); ok {
			if e = s.validateAliasTarget(ctx, tx, v); e != nil {
				return e
			}
		}
		switch v := value.(type) {
		case ModelAlias:
			v.Revision = rev + 1
			value = v
		case ModelSource:
			v.Revision = rev + 1
			value = v
		}
		if e = s.putDomain(ctx, tx, field, value); e != nil {
			return e
		}
		proof, _ := intent(value)
		key := "model_alias_proofs"
		if field == "ModelSources" {
			key = "model_source_proofs"
		}
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{key: map[string]string{id: proof}})
		if e != nil {
			return e
		}
		return s.domainTx(ctx, tx, field, id, out)
	})
}
func (s *Store) validateAliasTarget(ctx context.Context, tx *sql.Tx, v ModelAlias) error {
	target, e := DecodeModelTarget(v.Target)
	if e != nil {
		return e
	}
	switch target.Kind {
	case "base":
		var m BaseModelInstall
		return s.domainTx(ctx, tx, "BaseModels", target.ID, &m)
	case "speaker":
		var version ModelVersion
		return s.domainTx(ctx, tx, "Versions", target.ID, &version)
	}
	return nil
}
func (s *Store) PutModelAlias(ctx context.Context, op string, expected int64, v ModelAlias) (out ModelAlias, e error) {
	v.Revision = 0
	if v.ID == "" {
		v.ID = ModelReferenceID(s.workspace, "alias", v.Name)
	}
	if !validModelAlias(v) {
		return out, contracts.Fail("invalid_request")
	}
	e = s.modelRefWrite(ctx, op, expected, "ModelAliases", v.ID, v.Name, v, &out)
	return
}
func (s *Store) PutModelSource(ctx context.Context, op string, expected int64, v ModelSource) (out ModelSource, e error) {
	v.Revision = 0
	if v.ID == "" {
		v.ID = ModelReferenceID(s.workspace, "source", v.Name)
	}
	if !validModelSource(v) {
		return out, contracts.Fail("invalid_request")
	}
	e = s.modelRefWrite(ctx, op, expected, "ModelSources", v.ID, v.Name, v, &out)
	return
}
func (s *Store) RemoveModelAlias(ctx context.Context, op string, expected int64, id string) (ModelAlias, error) {
	v, e := s.ModelAlias(ctx, id)
	if e != nil {
		return v, e
	}
	v.State = "deleted"
	return s.PutModelAlias(ctx, op, expected, v)
}
func (s *Store) RemoveModelSource(ctx context.Context, op string, expected int64, id string) (ModelSource, error) {
	v, e := s.ModelSource(ctx, id)
	if e != nil {
		return v, e
	}
	v.State = "deleted"
	return s.PutModelSource(ctx, op, expected, v)
}
func (s *Store) validateModelReferenceState(ctx context.Context, tx *sql.Tx, r Records) error {
	proofs := map[string]map[string]string{"model_alias_proofs": {}, "model_source_proofs": {}}
	rows, e := tx.QueryContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			break
		}
		var m map[string]json.RawMessage
		if e = json.Unmarshal([]byte(raw), &m); e != nil {
			break
		}
		for k, p := range proofs {
			if raw, ok := m[k]; ok {
				var got map[string]string
				if e = json.Unmarshal(raw, &got); e != nil {
					break
				}
				for id, d := range got {
					p[id] = d
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
	for _, v := range r.ModelAliases {
		if !validModelAlias(v) || v.Revision < 1 {
			return contracts.Fail("invalid_request")
		}
		if e = s.validateAliasTarget(ctx, tx, v); e != nil {
			return contracts.Fail("invalid_request")
		}
		d, _ := intent(v)
		if proofs["model_alias_proofs"][v.ID] != d {
			return contracts.Fail("invalid_request")
		}
		delete(proofs["model_alias_proofs"], v.ID)
	}
	for _, v := range r.ModelSources {
		if !validModelSource(v) || v.Revision < 1 {
			return contracts.Fail("invalid_request")
		}
		d, _ := intent(v)
		if proofs["model_source_proofs"][v.ID] != d {
			return contracts.Fail("invalid_request")
		}
		delete(proofs["model_source_proofs"], v.ID)
	}
	if len(proofs["model_alias_proofs"])+len(proofs["model_source_proofs"]) > 0 {
		return contracts.Fail("invalid_request")
	}
	return nil
}

type SpeakerModelLineage struct {
	Version      ModelVersion       `json:"version"`
	Model        Model              `json:"model"`
	Run          TrainingRun        `json:"run"`
	Dataset      Dataset            `json:"dataset"`
	Artifacts    []ModelArtifact    `json:"artifacts"`
	Associations []ModelAssociation `json:"associations"`
}

func (s *Store) Artifact(ctx context.Context, id string) (out Artifact, e error) {
	e = s.readOne(ctx, "Artifacts", id, &out)
	return
}
func (s *Store) SpeakerModelVersion(ctx context.Context, id string) (out SpeakerModelLineage, e error) {
	if !contracts.ValidID(id) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	if e = s.domainTx(ctx, tx, "Versions", id, &out.Version); e != nil {
		return out, sanitize(e)
	}
	for _, x := range []struct {
		field, id string
		out       any
	}{{"Models", out.Version.ModelID, &out.Model}, {"Runs", out.Version.RunID, &out.Run}, {"Datasets", out.Version.DatasetID, &out.Dataset}} {
		if e = s.domainTx(ctx, tx, x.field, x.id, x.out); e != nil {
			return out, sanitize(e)
		}
	}
	r, e := s.readRecords(ctx, tx, "ModelArtifacts", "version_id", id)
	if e != nil {
		return out, sanitize(e)
	}
	out.Artifacts = []ModelArtifact{}
	for _, a := range r.ModelArtifacts {
		if a.VersionID == id {
			out.Artifacts = append(out.Artifacts, a)
		}
	}
	r, e = s.readRecords(ctx, tx, "ModelAssociations", "model_id", out.Model.ID)
	if e != nil {
		return out, sanitize(e)
	}
	out.Associations = []ModelAssociation{}
	for _, a := range r.ModelAssociations {
		if a.ModelID == out.Model.ID {
			out.Associations = append(out.Associations, a)
		}
	}
	return out, nil
}
