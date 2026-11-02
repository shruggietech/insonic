// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
)

const answerArtifactKind = "answer-evidence-context"

type answerPointer struct {
	Version   string `json:"version"`
	Workspace string `json:"workspace_id"`
	Key       string `json:"key"`
	ID        string `json:"publication_id"`
	Digest    string `json:"digest"`
}

func (p answerPointer) valid(workspace, key string) bool {
	return p.Version == contracts.Version && p.Workspace == workspace && p.Key == key && answers.ValidDigest(p.Key) && answers.ValidDigest(p.Digest) && contracts.ValidID(p.ID)
}

func (a *App) saveAnswerPackage(ctx context.Context, p answers.Package, expected int64) error {
	raw, err := json.Marshal(p)
	if err != nil || len(raw) > answers.MaxPackageBytes {
		return contracts.Fail("output_limit")
	}
	if e := schemas.ValidateAnswerPackage(raw); e != nil {
		return e
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	service, err := a.artifactServiceContext(ctx)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(a.Workspace.Control, "answer-context-*.json")
	if err != nil {
		return contracts.Fail("unavailable")
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return contracts.Fail("unavailable")
	}
	if err = f.Close(); err != nil {
		return contracts.Fail("unavailable")
	}
	if _, err = service.PublishVerified(ctx, p.ID, name, answerArtifactKind, digest, int64(len(raw))); err != nil {
		return err
	}
	// Every immutable historical package is retained independently of the
	// replaceable cache pointer, including a losing concurrent cache writer.
	if err = a.Catalog.ArtifactReference(ctx, p.ID, p.ID, false); err != nil {
		return err
	}
	if !answerCacheable(p) {
		return nil
	}
	pointer := answerPointer{Version: contracts.Version, Workspace: p.WorkspaceID, Key: p.Key, ID: p.ID, Digest: digest}
	value, _ := json.Marshal(pointer)
	setting := "answer-context:" + p.Key
	_, err = a.Catalog.PutNamedSetting(ctx, contracts.ID(), setting, expected, value)
	if err == nil {
		return nil
	}
	// A lost response or CAS race is reconciled by reading actual persisted
	// authority. Do not blindly overwrite a concurrent compatible pointer.
	actual, _, readErr := a.Catalog.NamedSetting(ctx, setting)
	if readErr != nil {
		return err
	}
	var winner answerPointer
	if contracts.DecodeExplore(actual, &winner) == nil && winner.valid(p.WorkspaceID, p.Key) {
		if winner.ID == p.ID && winner.Digest == digest {
			return nil
		}
		// The other pointer does not govern this retained publication. A later
		// cache reader independently verifies its bytes and semantic binding.
		return nil
	}
	return err
}

func (a *App) readAnswerPackage(ctx context.Context, id, digest string) (answers.Package, error) {
	p, _, err := a.readAnswerPackageBound(ctx, id, digest, answers.MaxPackageBytes)
	return p, err
}
func (a *App) readAnswerPackageBound(ctx context.Context, id, digest string, maxBytes int) (out answers.Package, size int, err error) {
	if !contracts.ValidID(id) || maxBytes < 1 {
		return out, 0, contracts.Fail("invalid_request")
	}
	publication, err := a.Catalog.Publication(ctx, id)
	if err != nil {
		return out, 0, err
	}
	if publication.Kind != answerArtifactKind || publication.Size < 1 || publication.Size > int64(min(maxBytes, answers.MaxPackageBytes)) || digest != "" && publication.Digest != digest {
		return out, 0, contracts.Condition("conflict", "answer_package_integrity")
	}
	service, err := a.artifactServiceContext(ctx)
	if err != nil {
		return out, 0, err
	}
	materialized, err := service.MaterializeBound(ctx, id, int64(min(maxBytes, answers.MaxPackageBytes)))
	if err != nil {
		return out, int(publication.Size), err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = service.Release(cleanup, id, materialized.Lease.ID)
	}()
	f, err := os.Open(materialized.Path)
	if err != nil {
		return out, 0, contracts.Fail("unavailable")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, int64(min(maxBytes, answers.MaxPackageBytes))+1))
	size = len(raw)
	if err != nil || size > min(maxBytes, answers.MaxPackageBytes) {
		return out, size, contracts.Fail("output_limit")
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != publication.Digest || schemas.ValidateAnswerPackage(raw) != nil || contracts.DecodeExplore(raw, &out) != nil || out.Kind != answerArtifactKind || out.ID != id || out.Version != contracts.Version || out.WorkspaceID != a.Workspace.Config.WorkspaceID || !answers.ValidDigest(out.Key) || !answers.ValidDigest(out.Observation.SourceEpoch) {
		return out, size, contracts.Condition("conflict", "answer_package_integrity")
	}
	normalized, e := answers.Normalize(out.Request)
	if e != nil || !reflect.DeepEqual(normalized, out.Request) || len(out.Items) > out.Request.Budget.Items || answers.ValidateContext(out.Context, out.Items, out.Request.Budget) != nil {
		return out, size, contracts.Condition("conflict", "answer_package_integrity")
	}
	seen := map[string]bool{}
	for _, item := range out.Items {
		ref := item.Reference
		if answers.ValidateReference(ref) != nil || ref.WorkspaceID != out.WorkspaceID || seen[item.ID] || item.ID != "cue:"+ref.RecordingID+":"+ref.DocumentDigest+":"+ref.CueID {
			return out, size, contracts.Condition("conflict", "answer_package_integrity")
		}
		seen[item.ID] = true
	}
	return out, size, nil
}

func (a *App) inspectAnswer(parent context.Context, id string) (out answerResult, err error) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	defer func() {
		out.Usage.ElapsedMS = time.Since(started).Milliseconds()
		if ctx.Err() != nil {
			err = contracts.Fail("cancelled")
		}
	}()
	out.Diagnostics = []string{}
	out.Freshness = "unknown"
	var size int
	out.Package, size, err = a.readAnswerPackageBound(ctx, id, "", answers.MaxPackageBytes)
	out.Usage.SourceBytes = size
	out.Usage.Rounds = 1
	if err != nil {
		return out, err
	}
	epoch, err := a.Catalog.AnswerEpoch(ctx, out.Package.Request.Filters.MediaIDs, 100000, 8<<20)
	out.Usage.Rows = epoch.Rows
	out.Usage.SourceBytes += epoch.Bytes
	out.Usage.Rounds++
	if err != nil || !epoch.Complete {
		out.Diagnostics = append(out.Diagnostics, "source_identity_unverified")
		return out, nil
	}
	out.Freshness = "current"
	if epoch.Digest != out.Package.Observation.SourceEpoch {
		out.Freshness = "stale"
		out.Diagnostics = append(out.Diagnostics, "source_authority_changed")
	}
	return out, nil
}
func (a *App) validateAnswer(parent context.Context, envelope answers.Envelope) error {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	view, err := a.inspectAnswer(ctx, envelope.PackageID)
	if err != nil {
		return err
	}
	if view.Freshness != "current" {
		return contracts.Condition("conflict", "source_authority_changed")
	}
	if err = answers.ValidateEnvelope(envelope, view.Package); err != nil {
		return err
	}
	// Resolve citations themselves as well as the saved package epoch. Importing
	// a correctly hashed artifact cannot authorize fabricated source text.
	resolved := map[string]answers.Item{}
	remaining := 8 << 20
	for _, claim := range envelope.Claims {
		for _, cite := range claim.Citations {
			item, ok := resolved[cite.EvidenceID]
			if !ok {
				if len(resolved) >= 128 || remaining < 1 {
					return contracts.Fail("input_limit")
				}
				page, e := a.Catalog.AnswerCandidates(ctx, catalog.AnswerPageRequest{RecordingID: cite.Reference.RecordingID, CueID: cite.Reference.CueID, Limit: 2, MaxBytes: remaining})
				if e != nil {
					return e
				}
				remaining -= page.Bytes
				if page.LimitReason != "" || len(page.Items) != 1 || !reflect.DeepEqual(page.Items[0].Reference, cite.Reference) {
					return contracts.Condition("conflict", "citation_authority")
				}
				item = page.Items[0]
				resolved[cite.EvidenceID] = item
			}
			if !strings.Contains(item.Text, cite.Quote) {
				return contracts.Condition("conflict", "citation_authority")
			}
		}
	}
	after, err := a.Catalog.AnswerEpoch(ctx, view.Package.Request.Filters.MediaIDs, 100000, 8<<20)
	if err != nil {
		return err
	}
	if !after.Complete || after.Digest != view.Package.Observation.SourceEpoch {
		return contracts.Condition("conflict", "source_authority_changed")
	}
	return nil
}
func (a *App) answersDispatch(ctx context.Context, req contracts.Request) (any, error) {
	switch req.Operation {
	case "answers.request":
		work, err := a.requestAnswer(ctx, req)
		return workView(work), err
	case "answers.retry":
		if len(req.Data) > 0 {
			return nil, contracts.Fail("invalid_request")
		}
		work, err := a.retryAnswer(ctx, req.RequestID, req.ItemID)
		return workView(work), err
	case "answers.prepare":
		var r answers.Request
		if contracts.DecodeExplore(req.Data, &r) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return a.prepareAnswer(ctx, r)
	case "answers.show", "answers.export":
		if len(req.Data) > 0 {
			return nil, contracts.Fail("invalid_request")
		}
		pub, err := a.Catalog.Publication(ctx, req.ItemID)
		if err != nil {
			return nil, err
		}
		if pub.Kind == answers.GroundedKind {
			return a.inspectGroundedAnswer(ctx, req.ItemID)
		}
		return a.inspectAnswer(ctx, req.ItemID)
	case "answers.validate":
		var e answers.Envelope
		if contracts.DecodeExplore(req.Data, &e) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		if err := a.validateAnswer(ctx, e); err != nil {
			return nil, err
		}
		return map[string]any{"valid": true, "package_id": e.PackageID, "validation": "current-source-citation-authority;semantic-entailment-not-evaluated"}, nil
	}
	return nil, contracts.Fail("invalid_request")
}
