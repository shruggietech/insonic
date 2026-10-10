// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
)

const portableFilePrefix = "<portable-file:"

// PortableFile identifies excluded executable bytes without retaining a host path.
func PortableFile(digest string) string { return portableFilePrefix + digest + ">" }
func PortableFileDigest(value string) (string, bool) {
	if !strings.HasPrefix(value, portableFilePrefix) || !strings.HasSuffix(value, ">") {
		return "", false
	}
	d := strings.TrimSuffix(strings.TrimPrefix(value, portableFilePrefix), ">")
	return d, digestPattern.MatchString(d)
}

func portableProcessingPayload(w Work) (json.RawMessage, bool, error) {
	switch w.Kind {
	case "recordings.process", "models.ensure", "models.train", "recordings.match", "portable.configuration":
	default:
		return w.Payload, false, nil
	}
	var value map[string]any
	if strict(w.Payload, &value) != nil {
		return nil, false, contracts.Fail("invalid_request")
	}
	if w.Kind == "models.ensure" {
		delete(value, "tools")
	}
	// Frozen matching copies are transient inputs. Preserve only their accepted
	// digest, which gates reconstruction from retained current authority.
	if w.Kind == "recordings.match" {
		if snapshot, ok := value["snapshot"].(map[string]any); ok {
			if d, ok := snapshot["digest"].(string); ok && digestPattern.MatchString(d) {
				value["snapshot"] = map[string]any{"digest": d}
			}
		}
	}
	paths := map[string]string{}
	var collect func(any)
	collect = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if p, ok := x["path"].(string); ok && p != "" {
				if d, ok := x["sha256"].(string); ok && digestPattern.MatchString(d) {
					paths[p] = PortableFile(d)
				}
			}
			if p, ok := x["executable"].(string); ok && p != "" {
				if d, ok := x["executable_sha256"].(string); ok && digestPattern.MatchString(d) {
					paths[p] = PortableFile(d)
				}
			}
			for _, item := range x {
				collect(item)
			}
		case []any:
			for _, item := range x {
				collect(item)
			}
		}
	}
	collect(value)
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if len(ordered[i]) == len(ordered[j]) {
			return ordered[i] < ordered[j]
		}
		return len(ordered[i]) > len(ordered[j])
	})
	var unbound bool
	var visit func(any, string)
	visit = func(v any, location string) {
		switch x := v.(type) {
		case map[string]any:
			for key, item := range x {
				transientRecognition := strings.HasSuffix(location, "/recognition") && !strings.HasPrefix(location, "/options/parameters")
				if key == "hints" && transientRecognition && w.Kind != "portable.configuration" {
					delete(x, key)
					continue
				}
				if key == "context_digest" && transientRecognition && w.Kind != "portable.configuration" {
					delete(x, key)
					continue
				}
				if key == "context" && location == "/election" && w.Kind != "portable.configuration" {
					if c, ok := item.(map[string]any); ok {
						if _, compiled := c["joined"]; compiled {
							x[key] = map[string]any{}
							continue
						}
					}
				}
				if p, ok := item.(string); ok {
					if replacement, ok := paths[p]; ok {
						x[key] = replacement
					} else if portableLocatorKey(key) && nonportableArgument(p) {
						unbound = true
					}
				}
				if key == "arguments" {
					if args, ok := item.([]any); ok {
						for i, arg := range args {
							if text, ok := arg.(string); ok {
								for _, path := range ordered {
									text = strings.ReplaceAll(text, path, paths[path])
								}
								if nonportableAdapterArgument(text) {
									unbound = true
								}
								args[i] = text
							}
						}
					}
				}
				visit(item, location+"/"+key)
			}
		case []any:
			for _, item := range x {
				visit(item, location)
			}
		}
	}
	visit(value, "")
	if unbound {
		return nil, false, &contracts.Error{Code: "unsupported_capability", Message: "Portable capture requires every local adapter argument or parameter dependency to have an explicit executable or support-file SHA256 identity. Declare the dependency before retrying capture."}
	}
	payload, e := json.Marshal(value)
	if e != nil {
		return nil, false, e
	}
	before, _ := intent(w.Payload)
	after, _ := intent(json.RawMessage(payload))
	return payload, before != after, nil
}

func portableLocatorKey(key string) bool {
	switch key {
	case "path", "source_path", "input_path", "output_path", "manifest_path", "input_directory", "output_directory", "base_model_directory", "scratch_directory":
		return true
	}
	return false
}

func nonportableArgument(argument string) bool {
	for _, part := range strings.Fields(argument) {
		if i := strings.IndexByte(part, '='); i >= 0 {
			part = part[i+1:]
		}
		part = strings.Trim(part, "\"'")
		if strings.HasPrefix(part, "/") || strings.HasPrefix(part, `\\`) || len(part) > 2 && part[1] == ':' && (part[2] == '/' || part[2] == '\\') {
			return true
		}
	}
	return false
}

func nonportableAdapterArgument(argument string) bool {
	// A standalone Windows slash switch has no path separator, extension or
	// value. Keep those ordinary flags while rejecting explicit path values.
	if strings.HasPrefix(argument, "/") && len(argument) > 1 && !strings.ContainsAny(argument[1:], "/\\.:= \t\r\n") {
		for _, r := range argument[1:] {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
				return nonportableArgument(argument)
			}
		}
		return false
	}
	return nonportableArgument(argument)
}

func portableProcessingProofID(w Work) string {
	pd, _ := intent(w.Payload)
	return operationID("portable-processing-input", w.ID, pd)
}

func portableProcessingAcceptanceDigest(w Work, initial, adapter string) string {
	pd, _ := intent(w.Payload)
	d, _ := intent([]string{"portable-processing-input", w.ID, w.Kind, initial, pd, adapter})
	return d
}

// PortableWorkInput reports a native transformation proof independently of the
// changing work phase and claim journal. It never accepts a caller marker alone.
func (s *Store) PortableWorkInput(ctx context.Context, w Work) (bool, string, error) {
	var raw, accepted string
	e := s.db.QueryRowContext(ctx, s.query("SELECT result,digest FROM operation_receipt WHERE workspace_id=? AND id=?"), s.workspace, portableProcessingProofID(w)).Scan(&raw, &accepted)
	if e == sql.ErrNoRows {
		return false, "", nil
	}
	if e != nil {
		return false, "", sanitize(e)
	}
	var proof struct {
		ID      string `json:"work_id"`
		Adapter string `json:"portable_adapter_digest"`
		Input   struct {
			Initial string `json:"initial_digest"`
			Payload string `json:"payload_digest"`
		} `json:"portable_input_proof"`
	}
	var initial string
	if json.Unmarshal([]byte(raw), &proof) != nil {
		return false, "", contracts.Fail("invalid_request")
	}
	if e = s.db.QueryRowContext(ctx, s.query("SELECT digest FROM operation_receipt WHERE workspace_id=? AND id=?"), s.workspace, w.ID).Scan(&initial); e != nil {
		return false, "", sanitize(e)
	}
	pd, _ := intent(w.Payload)
	if proof.ID != w.ID || proof.Input.Initial != initial || proof.Input.Payload != pd || accepted != portableProcessingAcceptanceDigest(w, initial, proof.Adapter) {
		return false, "", contracts.Fail("invalid_request")
	}
	return true, proof.Adapter, nil
}

func nonportableAcquisition(w Work) bool {
	switch w.Kind {
	case "media.import", "library.import", "transcript.import", "models.register", "models.acquire":
	default:
		return false
	}
	var value any
	if strict(w.Payload, &value) != nil {
		return false
	}
	var visit func(any) bool
	visit = func(v any) bool {
		switch x := v.(type) {
		case map[string]any:
			for key, item := range x {
				if key == "source" || key == "transcript" || key == "subtitle" || key == "source_path" || key == "manifest_path" {
					if text, ok := item.(string); ok && text != "" && text != "<accepted>" && !strings.HasPrefix(text, "https://") && !strings.HasPrefix(text, "http://") {
						return true
					}
				}
				if visit(item) {
					return true
				}
			}
		case []any:
			for _, item := range x {
				if visit(item) {
					return true
				}
			}
		}
		return false
	}
	return visit(value)
}

// PortableSnapshot transforms only a disposable validated catalog. Source
// processing stays untouched. New receipts prove that excluded local scratch
// cannot be resumed through copied original paths after restoration.
func PortableSnapshot(ctx context.Context, snap Snapshot) (Snapshot, error) {
	needed := false
	for _, w := range snap.Records.Works {
		_, processing, e := portableProcessingPayload(w)
		if e != nil {
			return Snapshot{}, e
		}
		if nonportableAcquisition(w) || processing {
			needed = true
			break
		}
	}
	for _, p := range snap.Records.Pipelines {
		_, changed, e := portableProcessingPayload(Work{Kind: "portable.configuration", Payload: p.Configuration})
		if e != nil {
			return Snapshot{}, e
		}
		needed = needed || changed
	}
	for _, r := range snap.Records.Runs {
		_, changed, e := portableProcessingPayload(Work{Kind: "portable.configuration", Payload: r.Options})
		if e != nil {
			return Snapshot{}, e
		}
		needed = needed || changed
	}
	if !needed {
		return snap, nil
	}
	ctx = context.WithValue(ctx, transferContextKey{}, nil)
	dir, e := os.MkdirTemp("", "insonic-portable-catalog-*")
	if e != nil {
		return Snapshot{}, contracts.Fail("unavailable")
	}
	defer os.RemoveAll(dir)
	s, e := OpenSQLite(ctx, filepath.Join(dir, "catalog.sqlite"), snap.WorkspaceID)
	if e != nil {
		return Snapshot{}, e
	}
	defer s.Close()
	if e = s.Restore(ctx, snap); e != nil {
		return Snapshot{}, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		for _, p := range snap.Records.Pipelines {
			payload, changed, e := portableProcessingPayload(Work{Kind: "portable.configuration", Payload: p.Configuration})
			if e != nil {
				return e
			}
			if !changed {
				continue
			}
			previous, _ := intent(p)
			p.Configuration = payload
			if e = s.putDomain(ctx, tx, "Pipelines", p); e != nil {
				return e
			}
			proof, _ := intent(p)
			if _, e = s.accept(ctx, tx, operationID("portable-pipeline", snap.Digest, p.ID), hash([]byte(previous+":"+proof)), rev, map[string]any{"pipeline_proofs": map[string]string{p.ID: proof}, "portable_pipeline_input": map[string]string{"previous_digest": previous, "configuration_digest": proof}}); e != nil {
				return e
			}
			rev++
		}
		for _, r := range snap.Records.Runs {
			payload, changed, e := portableProcessingPayload(Work{Kind: "portable.configuration", Payload: r.Options})
			if e != nil {
				return e
			}
			if !changed {
				continue
			}
			previous, _ := intent(r)
			r.Options = payload
			if e = s.putDomain(ctx, tx, "Runs", r); e != nil {
				return e
			}
			proof, _ := intent(r)
			if _, e = s.accept(ctx, tx, operationID("portable-training-run", snap.Digest, r.ID), hash([]byte(previous+":"+proof)), rev, map[string]any{"portable_training_run_input": map[string]string{"run_id": r.ID, "previous_digest": previous, "record_digest": proof}}); e != nil {
				return e
			}
			rev++
		}
		for _, original := range snap.Records.Works {
			payload, processing, e := portableProcessingPayload(original)
			if e != nil {
				return e
			}
			acquisition := nonportableAcquisition(original)
			if !acquisition && !processing {
				continue
			}
			var w Work
			if e := s.domainTx(ctx, tx, "Works", original.ID, &w); e != nil {
				return e
			}
			var initial string
			if e := s.row(ctx, tx, "SELECT digest FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, w.ID).Scan(&initial); e != nil {
				return e
			}
			proofID := operationID("portable-input", snap.Digest, w.ID)
			if acquisition {
				w.Payload, _ = json.Marshal(map[string]string{"portable_input": "source-required", "input_digest": initial, "proof_receipt_id": proofID})
				if w.State == "pending" || w.State == "running" || w.State == "interrupted" {
					w.State = "failed"
					w.Error = "operation_failed"
				}
				w.LeaseUntil = 0
				w.Phase = "portable-source-required"
			} else {
				w.Payload = payload
				if w.State == "running" {
					w.State = "interrupted"
				}
				w.LeaseUntil = 0
				proofID = portableProcessingProofID(w)
			}
			w.JournalReceiptID = proofID
			if e := s.putDomain(ctx, tx, "Works", w); e != nil {
				return e
			}
			pd, _ := intent(w.Payload)
			result := map[string]any{"work_id": w.ID, "work_digest": workJournalDigest(w), "portable_input_proof": map[string]string{"initial_digest": initial, "payload_digest": pd}}
			accepted := hash([]byte("portable-input:" + snap.Digest + ":" + w.ID))
			if processing {
				adapter := ""
				var input struct {
					Options struct {
						Adapter json.RawMessage `json:"adapter"`
					} `json:"options"`
				}
				json.Unmarshal(original.Payload, &input)
				if len(input.Options.Adapter) > 0 {
					var raw bytes.Buffer
					json.Compact(&raw, input.Options.Adapter)
					adapter = hash(raw.Bytes())
					result["portable_adapter_digest"] = adapter
				}
				accepted = portableProcessingAcceptanceDigest(w, initial, adapter)
			}
			if _, e := s.accept(ctx, tx, w.JournalReceiptID, accepted, rev, result); e != nil {
				return e
			}
			rev++
		}
		return s.validateLibraryState(ctx, tx, false)
	})
	if e != nil {
		return Snapshot{}, e
	}
	return s.Export(ctx)
}

func (s *Store) validatePortableProcessingInput(ctx context.Context, tx *sql.Tx, w Work, initial string) (bool, error) {
	var raw, accepted string
	e := s.row(ctx, tx, "SELECT result,digest FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, portableProcessingProofID(w)).Scan(&raw, &accepted)
	if e == sql.ErrNoRows {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	var proof struct {
		ID      string `json:"work_id"`
		Adapter string `json:"portable_adapter_digest"`
		Input   struct {
			Initial string `json:"initial_digest"`
			Payload string `json:"payload_digest"`
		} `json:"portable_input_proof"`
	}
	pd, _ := intent(w.Payload)
	if json.Unmarshal([]byte(raw), &proof) != nil || proof.ID != w.ID || proof.Input.Initial != initial || proof.Input.Payload != pd || accepted != portableProcessingAcceptanceDigest(w, initial, proof.Adapter) {
		return false, contracts.Fail("invalid_request")
	}
	return true, nil
}

func (s *Store) validatePortableWorkInput(ctx context.Context, tx *sql.Tx, w Work, initial string) error {
	var marker map[string]string
	if strict(w.Payload, &marker) != nil || len(marker) != 3 || marker["portable_input"] != "source-required" || marker["input_digest"] != initial || !contracts.ValidID(marker["proof_receipt_id"]) {
		return contracts.Fail("invalid_request")
	}
	var raw string
	if e := s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, marker["proof_receipt_id"]).Scan(&raw); e != nil {
		return e
	}
	var proof struct {
		ID    string `json:"work_id"`
		Input struct {
			Initial string `json:"initial_digest"`
			Payload string `json:"payload_digest"`
		} `json:"portable_input_proof"`
	}
	pd, _ := intent(w.Payload)
	if json.Unmarshal([]byte(raw), &proof) != nil || proof.ID != w.ID || proof.Input.Initial != initial || proof.Input.Payload != pd {
		return contracts.Fail("invalid_request")
	}
	return nil
}
