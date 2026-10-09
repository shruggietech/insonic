// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
)

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
		if nonportableAcquisition(w) {
			needed = true
			break
		}
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
		for _, original := range snap.Records.Works {
			if !nonportableAcquisition(original) {
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
			w.Payload, _ = json.Marshal(map[string]string{"portable_input": "source-required", "input_digest": initial, "proof_receipt_id": proofID})
			if w.State == "pending" || w.State == "running" || w.State == "interrupted" {
				w.State = "failed"
				w.Error = "operation_failed"
			}
			w.LeaseUntil = 0
			w.Phase = "portable-source-required"
			w.JournalReceiptID = proofID
			if e := s.putDomain(ctx, tx, "Works", w); e != nil {
				return e
			}
			pd, _ := intent(w.Payload)
			if _, e := s.accept(ctx, tx, w.JournalReceiptID, hash([]byte("portable-input:"+snap.Digest+":"+w.ID)), rev, map[string]any{"work_id": w.ID, "work_digest": workJournalDigest(w), "portable_input_proof": map[string]string{"initial_digest": initial, "payload_digest": pd}}); e != nil {
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
