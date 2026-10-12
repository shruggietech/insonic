// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"strings"
	"unicode"
	"unicode/utf8"
)

const importPresetSetting = "import-presets"

type ImportPreset struct {
	Kind     string          `json:"kind"`
	Version  string          `json:"schema_version"`
	ID       string          `json:"id"`
	Revision int64           `json:"revision,omitempty"`
	Name     string          `json:"name"`
	State    string          `json:"state,omitempty"`
	Options  json.RawMessage `json:"options"`
}
type importPresetRegistry struct {
	Kind    string         `json:"kind"`
	Version string         `json:"schema_version"`
	Items   []ImportPreset `json:"items"`
}

func validateImportPreset(p ImportPreset) error {
	if p.Kind != "import-preset" || p.Version != contracts.Version {
		return contracts.Fail("incompatible_version")
	}
	if !contracts.ValidID(p.ID) || !utf8.ValidString(p.Name) || strings.TrimSpace(p.Name) != p.Name || utf8.RuneCountInString(p.Name) < 1 || utf8.RuneCountInString(p.Name) > 128 || strings.IndexFunc(p.Name, unicode.IsControl) >= 0 || p.Revision < 0 || p.State != "" && p.State != "active" && p.State != "deleted" || !nonsecret(p.Options) {
		return contracts.Fail("invalid_request")
	}
	// The shared import option schema enforces the same strict fields for saved choices.
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: "media.import"}
	request.Data, _ = json.Marshal(map[string]any{"kind": "import-manifest", "schema_version": contracts.Version, "defaults": p.Options, "items": []any{map[string]any{"source": "validation"}}})
	raw, _ := json.Marshal(request)
	if schemas.ValidateRequest(raw) != nil {
		return contracts.Fail("invalid_request")
	}
	var options struct {
		Preset           string             `json:"preset"`
		Destination      *workspace.Profile `json:"destination"`
		TimezoneImplicit bool               `json:"timezone_implicit"`
		Digest           string             `json:"diarization_model_digest"`
	}
	if json.Unmarshal(p.Options, &options) != nil || options.Preset != "" || options.TimezoneImplicit || options.Digest != "" {
		return contracts.Fail("invalid_request")
	}
	if options.Destination != nil {
		if _, e := profileRecord("storage", *options.Destination); e != nil {
			return e
		}
	}
	return nil
}
func decodeImportPresets(raw []byte, maxRevision int64) (importPresetRegistry, error) {
	r := importPresetRegistry{Kind: "import-preset-registry", Version: contracts.Version, Items: []ImportPreset{}}
	if len(raw) == 0 {
		return r, nil
	}
	if len(raw) > 1<<20 || strict(raw, &r) != nil || len(r.Items) > 1000 {
		return r, contracts.Fail("invalid_request")
	}
	if r.Kind != "import-preset-registry" || r.Version != contracts.Version {
		return r, contracts.Fail("incompatible_version")
	}
	ids, names := map[string]bool{}, map[string]bool{}
	for _, p := range r.Items {
		if err := validateImportPreset(p); err != nil {
			return r, err
		}
		if p.Revision < 1 || p.Revision > maxRevision || p.State == "" || ids[p.ID] || p.State == "active" && names[p.Name] {
			return r, contracts.Fail("invalid_request")
		}
		ids[p.ID] = true
		if p.State == "active" {
			names[p.Name] = true
		}
	}
	return r, nil
}
func (s *Store) ImportPresets(ctx context.Context) ([]ImportPreset, error) {
	raw, rev, e := s.NamedSetting(ctx, importPresetSetting)
	if e != nil {
		return nil, e
	}
	r, e := decodeImportPresets(raw, rev)
	return r.Items, e
}
func (s *Store) MutateImportPreset(ctx context.Context, op, action string, expected int64, p ImportPreset) (ImportPreset, error) {
	if !contracts.ValidID(op) || !contracts.ValidID(p.ID) || expected < 0 || action != "create" && action != "update" && action != "delete" || action == "create" && expected != 0 || action != "create" && expected < 1 {
		return p, contracts.Fail("invalid_request")
	}
	if action != "delete" {
		if e := validateImportPreset(p); e != nil {
			return p, e
		}
		if p.State == "deleted" {
			return p, contracts.Fail("invalid_request")
		}
	}
	digest, e := intent(map[string]any{"action": action, "expected_revision": expected, "preset": p})
	if e != nil {
		return p, e
	}
	var out ImportPreset
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		receipt, found, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if found {
			var frozen struct {
				Preset ImportPreset `json:"preset"`
			}
			if json.Unmarshal(receipt.Result, &frozen) != nil || frozen.Preset.ID != p.ID {
				return contracts.Fail("invalid_request")
			}
			out = frozen.Preset
			return nil
		}
		var raw string
		var settingRev int64
		e = s.row(ctx, tx, "SELECT value,revision FROM setting_revision WHERE workspace_id=? AND name=? ORDER BY revision DESC LIMIT 1", s.workspace, importPresetSetting).Scan(&raw, &settingRev)
		if e != nil && e != sql.ErrNoRows {
			return e
		}
		registry, e := decodeImportPresets([]byte(raw), settingRev)
		if e != nil {
			return e
		}
		index := -1
		for i, current := range registry.Items {
			if current.ID == p.ID {
				index = i
			}
			if action != "delete" && current.ID != p.ID && current.State == "active" && current.Name == p.Name {
				return contracts.Fail("conflict")
			}
		}
		if action == "create" {
			if index >= 0 || len(registry.Items) >= 1000 {
				return contracts.Fail("conflict")
			}
		} else {
			if index < 0 || registry.Items[index].State != "active" {
				return contracts.Fail("not_found")
			}
			if registry.Items[index].Revision != expected {
				return contracts.Condition("conflict", "revision_stale")
			}
		}
		out = p
		if action == "delete" {
			out = registry.Items[index]
			out.State = "deleted"
		} else {
			out.State = "active"
		}
		out.Revision = rev + 1
		if index < 0 {
			registry.Items = append(registry.Items, out)
		} else {
			registry.Items[index] = out
		}
		value, _ := json.Marshal(registry)
		value, e = canonical(value)
		if e != nil || len(value) > 1<<20 {
			return contracts.Fail("output_limit")
		}
		var options struct {
			Destination *workspace.Profile `json:"destination"`
		}
		json.Unmarshal(out.Options, &options)
		if action != "delete" && options.Destination != nil {
			record, e := profileRecord("storage", *options.Destination)
			if e != nil {
				return e
			}
			if _, e = s.registerProfileTx(ctx, tx, record); e != nil {
				return e
			}
		}
		if _, e = s.exec(ctx, tx, "INSERT INTO setting_revision(workspace_id,name,revision,value) VALUES(?,?,?,?)", s.workspace, importPresetSetting, rev+1, string(value)); e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"preset": out, "registry_digest": hash(value)})
		return e
	})
	return out, e
}
