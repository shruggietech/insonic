// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"io"
	"os"
	"reflect"
	"time"
)

type groundedAnswerView struct {
	Answer      answers.Grounded `json:"answer"`
	Accepted    bool             `json:"accepted"`
	Freshness   string           `json:"freshness"`
	Diagnostics []string         `json:"diagnostics"`
}

func (a *App) saveGroundedAnswer(ctx context.Context, workID, id string, raw []byte) error {
	if schemas.ValidateGroundedAnswer(raw) != nil {
		return contracts.Fail("invalid_request")
	}
	if len(raw) > answers.MaxPackageBytes {
		return contracts.Fail("output_limit")
	}
	service, err := a.artifactServiceContext(ctx)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(a.Workspace.Control, "grounded-answer-*.json")
	if err != nil {
		return contracts.Fail("unavailable")
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return contracts.Fail("unavailable")
	}
	if err = f.Close(); err != nil {
		return contracts.Fail("unavailable")
	}
	if _, err = service.PublishVerified(ctx, id, f.Name(), answers.GroundedKind, documentHash(raw), int64(len(raw))); err != nil {
		return err
	}
	return a.Catalog.ArtifactReference(ctx, id, workID, false)
}
func (a *App) readGroundedAnswer(ctx context.Context, id, digest string) (out answers.Grounded, err error) {
	pub, err := a.Catalog.Publication(ctx, id)
	if err != nil {
		return out, err
	}
	if pub.Kind != answers.GroundedKind || pub.Size < 1 || pub.Size > answers.MaxPackageBytes || digest != "" && pub.Digest != digest {
		return out, contracts.Fail("conflict")
	}
	service, err := a.artifactServiceContext(ctx)
	if err != nil {
		return out, err
	}
	materialized, err := service.MaterializeBound(ctx, id, answers.MaxPackageBytes)
	if err != nil {
		return out, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = service.Release(cleanup, id, materialized.Lease.ID)
	}()
	f, err := os.Open(materialized.Path)
	if err != nil {
		return out, contracts.Fail("unavailable")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, answers.MaxPackageBytes+1))
	if err != nil {
		return out, contracts.Fail("conflict")
	}
	return a.decodeGroundedAnswer(ctx, id, pub.Digest, pub.Size, raw)
}

// Both a durable checkpoint and a published object require the same bounded
// identity, context and citation validation before they can be accepted.
func (a *App) decodeGroundedAnswer(ctx context.Context, id, digest string, size int64, raw []byte) (out answers.Grounded, err error) {
	if len(raw) < 1 || len(raw) > answers.MaxPackageBytes || int64(len(raw)) != size || documentHash(raw) != digest || schemas.ValidateGroundedAnswer(raw) != nil || contracts.DecodeExplore(raw, &out) != nil {
		return out, contracts.Fail("conflict")
	}
	if out.Kind != answers.GroundedKind || out.Version != contracts.Version || out.ID != id || out.WorkspaceID != a.Workspace.Config.WorkspaceID || !contracts.ValidID(out.WorkID) || !answers.ValidDigest(out.SourceEpoch) || out.Answer.PackageID != out.PackageID || len(out.Attempts) > 2 {
		return out, contracts.Fail("conflict")
	}
	p, err := a.readAnswerPackage(ctx, out.PackageID, "")
	if err != nil {
		return out, err
	}
	if p.Observation.SourceEpoch != out.SourceEpoch || !reflect.DeepEqual(p.Request.Provider, out.Provider) || answers.ValidateSynthesis(out.Answer, p) != nil {
		return out, contracts.Fail("conflict")
	}
	return out, nil
}
func (a *App) inspectGroundedAnswer(parent context.Context, id string) (out groundedAnswerView, err error) {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	out.Freshness = "unknown"
	out.Diagnostics = []string{}
	out.Answer, err = a.readGroundedAnswer(ctx, id, "")
	if err != nil {
		return out, err
	}
	work, e := a.Catalog.Work(ctx, out.Answer.WorkID)
	var progress answerProgress
	if e == nil && work.Kind == "answers.request" && work.State == "succeeded" && json.Unmarshal(work.Result, &progress) == nil && progress.Accepted && progress.Candidate != nil && progress.AnswerID == id {
		pub, readErr := a.Catalog.Publication(ctx, id)
		out.Accepted = readErr == nil && progress.Candidate.Digest == pub.Digest && progress.Candidate.Size == pub.Size && progress.Candidate.PackageID == out.Answer.PackageID && progress.Candidate.SourceEpoch == out.Answer.SourceEpoch
	}
	if !out.Accepted {
		out.Diagnostics = append(out.Diagnostics, "answer_not_accepted")
		return out, nil
	}
	view, e := a.inspectAnswer(ctx, out.Answer.PackageID)
	if e != nil {
		out.Diagnostics = append(out.Diagnostics, "source_identity_unverified")
		return out, nil
	}
	out.Freshness = view.Freshness
	out.Diagnostics = append(out.Diagnostics, view.Diagnostics...)
	return out, nil
}
