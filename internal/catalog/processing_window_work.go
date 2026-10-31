// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/shruggietech/insonic/internal/contracts"
)

// ProcessingPlan contains a content-free frozen execution plan. Window output
// lives in verified temporary artifacts, not in work journals or plan JSON.
type ProcessingPlan struct {
	ID                  string          `json:"id"`
	WorkID              string          `json:"work_id"`
	MediaID             string          `json:"media_id"`
	SourceRevision      int64           `json:"source_revision"`
	SourceDigest        string          `json:"source_digest"`
	ExpectedRevision    int64           `json:"expected_revision"`
	DocumentDigest      string          `json:"document_digest"`
	ConfigurationDigest string          `json:"configuration_digest"`
	PlanDigest          string          `json:"plan_digest"`
	Plan                json.RawMessage `json:"plan"`
	Attempt             int64           `json:"attempt"`
	State               string          `json:"state"`
}

// ProcessingWindowCheckpoint preserves producer identity after retirement,
// while only ready checkpoints pin output bytes and are eligible for resume.
type ProcessingWindowCheckpoint struct {
	ID            string `json:"id"`
	PlanID        string `json:"plan_id"`
	WorkID        string `json:"work_id"`
	WindowID      string `json:"window_id"`
	Stage         string `json:"stage"`
	Attempt       int64  `json:"attempt"`
	PublicationID string `json:"publication_id"`
	OutputDigest  string `json:"output_digest"`
	OutputSize    int64  `json:"output_size"`
	State         string `json:"state"`
}

const MaxProcessingPlanBytes = 16 << 20

func ProcessingPlanID(workID, digest string) string {
	return operationID("processing-plan", workID, digest)
}
func ProcessingCheckpointID(planID, windowID, stage string) string {
	return operationID("processing-window", planID, windowID, stage)
}

// ProcessingWindowPublicationID includes the producing generation so an expired
// owner can never replace the bytes published by its successor.
func ProcessingWindowPublicationID(workID, planDigest, windowID string, generation int64) string {
	label := fmt.Sprintf("processing-window/%s/%s/%d", planDigest, windowID, generation)
	sum := sha256.Sum256([]byte(workID + "\x00" + label))
	sum[6] = sum[6]&15 | 64
	sum[8] = sum[8]&63 | 128
	id := hex.EncodeToString(sum[:16])
	return id[:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:]
}

// This projection deliberately avoids importing processing, which uses catalog.
// It checks identity and window membership at the durable authority boundary.
type processingPlanProjection struct {
	Kind        string `json:"kind"`
	Version     string `json:"schema_version"`
	PlanVersion int    `json:"plan_version"`
	Digest      string `json:"digest"`
	Identity    struct {
		SourceID            string `json:"source_id"`
		SourceDigest        string `json:"source_sha256"`
		SourceRevision      int64  `json:"source_revision"`
		DocumentDigest      string `json:"document_sha256"`
		DocumentRevision    int64  `json:"document_revision"`
		ConfigurationDigest string `json:"configuration_sha256"`
		ModelDigest         string `json:"model_sha256"`
		AdapterID           string `json:"adapter_id"`
		AdapterVersion      string `json:"adapter_version"`
		DevicePolicy        string `json:"device_policy"`
	} `json:"identity"`
	Options struct {
		CoreUS      int64  `json:"core_us"`
		ContextUS   *int64 `json:"context_us"`
		Concurrency int    `json:"concurrency"`
		Tracks      []struct {
			StreamIndex int    `json:"stream_index"`
			Channel     *int   `json:"channel"`
			StartSample int64  `json:"start_sample"`
			EndSample   *int64 `json:"end_sample"`
		} `json:"tracks"`
	} `json:"options"`
	Tracks []struct {
		StreamIndex       int    `json:"stream_index"`
		Channel           *int   `json:"channel"`
		SourceSampleRate  int64  `json:"source_sample_rate"`
		SourceSampleCount int64  `json:"source_sample_count"`
		StartNumerator    string `json:"start_numerator"`
		StartDenominator  string `json:"start_denominator"`
		StartSample       int64  `json:"start_sample"`
		EndSample         int64  `json:"end_sample"`
	} `json:"tracks"`
	Windows []struct {
		ID                 string `json:"id"`
		Ordinal            int    `json:"ordinal"`
		TrackOrdinal       int    `json:"track_ordinal"`
		StreamIndex        int    `json:"stream_index"`
		Channel            *int   `json:"channel"`
		SourceSampleRate   int64  `json:"source_sample_rate"`
		CoreStartSample    int64  `json:"core_start_sample"`
		CoreEndSample      int64  `json:"core_end_sample"`
		ContextStartSample int64  `json:"context_start_sample"`
		ContextEndSample   int64  `json:"context_end_sample"`
		MappedStartSample  int64  `json:"mapped_start_sample"`
		MappedEndSample    int64  `json:"mapped_end_sample"`
		Mapping            struct {
			StartNumerator   string `json:"start_numerator"`
			StartDenominator string `json:"start_denominator"`
			SampleCount      int64  `json:"sample_count"`
			SampleRate       int64  `json:"sample_rate"`
			StreamIndex      int    `json:"stream_index"`
			Channel          *int   `json:"channel"`
			Policy           string `json:"policy"`
		} `json:"mapping"`
	} `json:"windows"`
}

func processingPlanDocument(p ProcessingPlan) (processingPlanProjection, error) {
	var doc processingPlanProjection
	if len(p.Plan) == 0 || len(p.Plan) > MaxProcessingPlanBytes || !referenceMetadata(p.Plan) || strict(p.Plan, &doc) != nil {
		return doc, contracts.Fail("invalid_request")
	}
	var fields map[string]json.RawMessage
	if strict(p.Plan, &fields) != nil {
		return doc, contracts.Fail("invalid_request")
	}
	delete(fields, "digest")
	digest, err := intent(fields)
	if err != nil || digest != p.PlanDigest || doc.Digest != digest || doc.Kind != "processing-window-plan" || !contracts.SupportedDocumentVersion(doc.Version) || doc.PlanVersion != 1 {
		return doc, contracts.Fail("invalid_request")
	}
	i := doc.Identity
	if i.SourceID != p.MediaID || i.SourceDigest != p.SourceDigest || i.SourceRevision != p.SourceRevision || i.DocumentDigest != p.DocumentDigest || i.DocumentRevision != p.ExpectedRevision || i.ConfigurationDigest != p.ConfigurationDigest || len(doc.Tracks) < 1 || len(doc.Tracks) > 64 || len(doc.Windows) > 16384 {
		return doc, contracts.Fail("invalid_request")
	}
	if i.ModelDigest != "" && !digestPattern.MatchString(i.ModelDigest) || len(i.AdapterID) > 256 || len(i.AdapterVersion) > 256 || len(i.DevicePolicy) > 256 || doc.Options.CoreUS < 1_000_000 || doc.Options.CoreUS > 300_000_000 || doc.Options.ContextUS == nil || *doc.Options.ContextUS < 0 || *doc.Options.ContextUS > doc.Options.CoreUS/2 || doc.Options.Concurrency < 1 || doc.Options.Concurrency > 4 || len(doc.Options.Tracks) != 0 && len(doc.Options.Tracks) != len(doc.Tracks) {
		return doc, contracts.Fail("invalid_request")
	}
	ends := make([]int64, len(doc.Tracks))
	trackKeys := map[string]bool{}
	for n, t := range doc.Tracks {
		if len(t.StartNumerator) > 128 || len(t.StartDenominator) > 128 {
			return doc, contracts.Fail("invalid_request")
		}
		_, numeratorOK := new(big.Int).SetString(t.StartNumerator, 10)
		denominator, denominatorOK := new(big.Int).SetString(t.StartDenominator, 10)
		if !numeratorOK || !denominatorOK || denominator.Sign() <= 0 || t.StreamIndex < 0 || t.StreamIndex > 65535 || t.Channel != nil && (*t.Channel < 0 || *t.Channel > 63) || t.SourceSampleRate <= 0 || t.SourceSampleRate > 768000 || t.SourceSampleCount < 0 || t.SourceSampleCount > 604800*t.SourceSampleRate || t.StartSample < 0 || t.EndSample < t.StartSample || t.EndSample > t.SourceSampleCount {
			return doc, contracts.Fail("invalid_request")
		}
		trackKey, _ := intent([]any{t.StreamIndex, t.Channel})
		if trackKeys[trackKey] {
			return doc, contracts.Fail("invalid_request")
		}
		trackKeys[trackKey] = true
		if len(doc.Options.Tracks) != 0 {
			selection := doc.Options.Tracks[n]
			end := t.SourceSampleCount
			if selection.EndSample != nil {
				end = *selection.EndSample
			}
			if selection.StreamIndex != t.StreamIndex || (selection.Channel == nil) != (t.Channel == nil) || selection.Channel != nil && *selection.Channel != *t.Channel || selection.StartSample != t.StartSample || end != t.EndSample {
				return doc, contracts.Fail("invalid_request")
			}
		}
		ends[n] = t.StartSample
	}
	seen := map[string]bool{}
	for n, w := range doc.Windows {
		if !contracts.ValidID(w.ID) || seen[w.ID] || w.Ordinal != n || w.TrackOrdinal < 0 || w.TrackOrdinal >= len(doc.Tracks) {
			return doc, contracts.Fail("invalid_request")
		}
		seen[w.ID] = true
		t := doc.Tracks[w.TrackOrdinal]
		if w.StreamIndex != t.StreamIndex || w.SourceSampleRate != t.SourceSampleRate || (w.Channel == nil) != (t.Channel == nil) || w.Channel != nil && *w.Channel != *t.Channel || w.CoreStartSample != ends[w.TrackOrdinal] || w.CoreEndSample <= w.CoreStartSample || w.CoreEndSample > t.EndSample || w.ContextStartSample < 0 || w.ContextStartSample > w.CoreStartSample || w.ContextEndSample < w.CoreEndSample || w.ContextEndSample > t.SourceSampleCount || w.MappedStartSample < 0 || w.MappedEndSample < w.MappedStartSample {
			return doc, contracts.Fail("invalid_request")
		}
		left := (w.ContextStartSample*16000 + t.SourceSampleRate - 1) / t.SourceSampleRate
		right := max(left, w.ContextEndSample*16000/t.SourceSampleRate)
		mapping := w.Mapping
		if w.ContextStartSample < t.StartSample || w.ContextEndSample > t.EndSample || w.MappedStartSample != left || w.MappedEndSample != right || mapping.SampleRate != 16000 || mapping.SampleCount != right-left || mapping.StreamIndex != t.StreamIndex || (mapping.Channel == nil) != (t.Channel == nil) || mapping.Channel != nil && *mapping.Channel != *t.Channel || len(mapping.StartNumerator) > 128 || len(mapping.StartDenominator) > 128 || mapping.Policy != "canonical-source-samples;inward-window;continuous-resample;source-start-rational" {
			return doc, contracts.Fail("invalid_request")
		}
		numerator, _ := new(big.Int).SetString(t.StartNumerator, 10)
		denominator, _ := new(big.Int).SetString(t.StartDenominator, 10)
		origin := new(big.Rat).SetFrac(numerator, denominator)
		origin.Add(origin, new(big.Rat).SetFrac64(left, 16000))
		if mapping.StartNumerator != origin.Num().String() || mapping.StartDenominator != origin.Denom().String() {
			return doc, contracts.Fail("invalid_request")
		}
		ends[w.TrackOrdinal] = w.CoreEndSample
	}
	for n, t := range doc.Tracks {
		if ends[n] != t.EndSample {
			return doc, contracts.Fail("invalid_request")
		}
	}
	return doc, nil
}

func validProcessingPlan(p ProcessingPlan) bool {
	if !contracts.ValidID(p.ID) || !contracts.ValidID(p.WorkID) || !contracts.ValidID(p.MediaID) || p.ID != ProcessingPlanID(p.WorkID, p.PlanDigest) || !digestPattern.MatchString(p.SourceDigest) || !digestPattern.MatchString(p.PlanDigest) || !digestPattern.MatchString(p.ConfigurationDigest) || p.DocumentDigest != "" && !digestPattern.MatchString(p.DocumentDigest) || p.SourceRevision < 1 || p.ExpectedRevision < 0 || p.Attempt < 1 || p.State != "active" && p.State != "completed" && p.State != "invalidated" {
		return false
	}
	_, err := processingPlanDocument(p)
	return err == nil
}
func validProcessingCheckpoint(c ProcessingWindowCheckpoint) bool {
	return contracts.ValidID(c.ID) && contracts.ValidID(c.PlanID) && contracts.ValidID(c.WorkID) && contracts.ValidID(c.WindowID) && contracts.ValidID(c.PublicationID) && c.ID == ProcessingCheckpointID(c.PlanID, c.WindowID, c.Stage) && (c.Stage == "combined" || c.Stage == "recognition" || c.Stage == "diarization") && c.Attempt > 0 && c.OutputSize > 0 && digestPattern.MatchString(c.OutputDigest) && (c.State == "ready" || c.State == "retired")
}

func (s *Store) ProcessingPlan(ctx context.Context, id string) (p ProcessingPlan, err error) {
	err = s.readOne(ctx, "ProcessingPlans", id, &p)
	return
}
func (s *Store) ProcessingPlans(ctx context.Context, workID string) ([]ProcessingPlan, error) {
	if !contracts.ValidID(workID) {
		return nil, contracts.Fail("invalid_request")
	}
	rows, err := s.db.QueryContext(ctx, s.query("SELECT "+joinNames(domainNamed("ProcessingPlans"))+" FROM processing_plan WHERE workspace_id=? AND work_id=? ORDER BY id"), s.workspace, workID)
	if err != nil {
		return nil, sanitize(err)
	}
	defer rows.Close()
	var out []ProcessingPlan
	for rows.Next() {
		var p ProcessingPlan
		if err = scanProcessingDomain(rows, domainNamed("ProcessingPlans"), &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, sanitize(rows.Err())
}

func scanProcessingDomain(rows *sql.Rows, d domainTable, out any) error {
	cols := columns(d.typ())
	values := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for n := range values {
		ptrs[n] = &values[n]
	}
	if err := rows.Scan(ptrs...); err != nil {
		return sanitize(err)
	}
	return decodeDomainRow(cols, values, out)
}

func validProcessingWork(p ProcessingPlan, w Work) bool {
	var input struct {
		MediaID        string `json:"media_id"`
		SourceRevision int64  `json:"source_revision"`
		Expected       int64  `json:"expected_revision"`
	}
	return (w.Kind == "recordings.process" || w.Kind == "recordings.repair") && w.ID == p.WorkID && p.Attempt <= w.Generation && json.Unmarshal(w.Payload, &input) == nil && input.MediaID == p.MediaID && input.SourceRevision == p.SourceRevision && input.Expected == p.ExpectedRevision
}
func (s *Store) processingSourceAuthority(ctx context.Context, tx *sql.Tx, p ProcessingPlan) error {
	var entry LibraryEntry
	if err := s.domainTx(ctx, tx, "Library", p.MediaID, &entry); err != nil {
		return err
	}
	if entry.Revision != p.SourceRevision || entry.Digest != p.SourceDigest {
		return contracts.Condition("conflict", "revision_stale")
	}
	var recording Recording
	err := s.domainTx(ctx, tx, "Recordings", p.MediaID, &recording)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == sql.ErrNoRows && (p.ExpectedRevision != 0 || p.DocumentDigest != "") || err == nil && (recording.Revision != p.ExpectedRevision || recording.DocumentDigest != p.DocumentDigest) {
		return contracts.Condition("conflict", "revision_stale")
	}
	return nil
}

func (s *Store) CommitProcessingPlan(ctx context.Context, claim Work, p ProcessingPlan) (out ProcessingPlan, err error) {
	if !validProcessingPlan(p) || p.State != "active" || p.WorkID != claim.ID || p.Attempt != claim.Generation {
		return out, contracts.Fail("invalid_request")
	}
	err = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		w, e := s.workAuthority(ctx, tx, claim)
		if e != nil {
			return e
		}
		if !validProcessingWork(p, w) {
			return contracts.Fail("invalid_request")
		}
		if e = s.processingSourceAuthority(ctx, tx, p); e != nil {
			return e
		}
		var existing ProcessingPlan
		if e = s.domainTx(ctx, tx, "ProcessingPlans", p.ID, &existing); e == nil {
			candidate := p
			candidate.Attempt = existing.Attempt
			d, _ := intent(candidate)
			old, _ := intent(existing)
			if d != old {
				return contracts.Fail("conflict")
			}
			out = existing
			return nil
		} else if e != sql.ErrNoRows {
			return e
		}
		if e = s.insertRecords(ctx, tx, Records{ProcessingPlans: []ProcessingPlan{p}}); e != nil {
			return e
		}
		digest, _ := intent(p)
		_, e = s.accept(ctx, tx, operationID("processing-plan-accept", p.ID), digest, rev, map[string]any{"processing_plan_proofs": map[string]string{p.ID: digest}})
		out = p
		return e
	})
	return
}

func (s *Store) ProcessingWindowCheckpoints(ctx context.Context, planID string) ([]ProcessingWindowCheckpoint, error) {
	if !contracts.ValidID(planID) {
		return nil, contracts.Fail("invalid_request")
	}
	rows, err := s.db.QueryContext(ctx, s.query("SELECT "+joinNames(domainNamed("ProcessingCheckpoints"))+" FROM processing_window_checkpoint WHERE workspace_id=? AND plan_id=? ORDER BY window_id,stage"), s.workspace, planID)
	if err != nil {
		return nil, sanitize(err)
	}
	defer rows.Close()
	var out []ProcessingWindowCheckpoint
	for rows.Next() {
		var c ProcessingWindowCheckpoint
		if err = scanProcessingDomain(rows, domainNamed("ProcessingCheckpoints"), &c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, sanitize(rows.Err())
}

func processingWindowMember(p ProcessingPlan, c ProcessingWindowCheckpoint) bool {
	if p.ID != c.PlanID || p.WorkID != c.WorkID {
		return false
	}
	var doc processingPlanProjection
	if json.Unmarshal(p.Plan, &doc) != nil {
		return false
	}
	for _, w := range doc.Windows {
		if w.ID == c.WindowID {
			return true
		}
	}
	return false
}
func (s *Store) processingCheckpointPublication(ctx context.Context, tx *sql.Tx, c ProcessingWindowCheckpoint) error {
	p, err := s.publicationTx(ctx, tx, c.PublicationID)
	if err != nil {
		return err
	}
	if p.Kind != "processing-window" || p.Digest != c.OutputDigest || p.Size != c.OutputSize || c.State == "ready" && p.State != "available" {
		return contracts.Fail("conflict")
	}
	return nil
}
func (s *Store) CommitProcessingWindowCheckpoint(ctx context.Context, claim Work, c ProcessingWindowCheckpoint) (out ProcessingWindowCheckpoint, err error) {
	if !validProcessingCheckpoint(c) || c.State != "ready" || c.WorkID != claim.ID || c.Attempt != claim.Generation {
		return out, contracts.Fail("invalid_request")
	}
	err = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		w, e := s.workAuthority(ctx, tx, claim)
		if e != nil {
			return e
		}
		var plan ProcessingPlan
		if e = s.domainTx(ctx, tx, "ProcessingPlans", c.PlanID, &plan); e != nil {
			return e
		}
		if plan.State != "active" || !validProcessingWork(plan, w) || !processingWindowMember(plan, c) {
			return contracts.Fail("conflict")
		}
		if e = s.processingSourceAuthority(ctx, tx, plan); e != nil {
			return e
		}
		if e = s.processingCheckpointPublication(ctx, tx, c); e != nil {
			return e
		}
		digest, _ := intent(c)
		_, ok, e := s.replay(ctx, tx, operationID("processing-window-accept", c.ID), digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "ProcessingCheckpoints", c.ID, &out)
		}
		if e = s.insertRecords(ctx, tx, Records{ProcessingCheckpoints: []ProcessingWindowCheckpoint{c}}); e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, operationID("processing-window-accept", c.ID), digest, rev, map[string]any{"processing_checkpoint_proofs": map[string]string{c.ID: digest}})
		out = c
		return e
	})
	return
}

func processingRecordProof(v any) string {
	digest, _ := intent(v)
	return digest
}
