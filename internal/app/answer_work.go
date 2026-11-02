// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"reflect"
	"time"
)

type AnswerWorkflowRequest struct {
	Request          answers.Request             `json:"request"`
	Configuration    *assistance.SynthesisConfig `json:"configuration,omitempty"`
	PipelineID       string                      `json:"pipeline_id,omitempty"`
	PipelineRevision int64                       `json:"pipeline_revision,omitempty"`
}
type answerPayload struct {
	Request          answers.Request            `json:"request"`
	Configuration    assistance.SynthesisConfig `json:"configuration"`
	PipelineID       string                     `json:"pipeline_id,omitempty"`
	PipelineRevision int64                      `json:"pipeline_revision,omitempty"`
	RequestDigest    string                     `json:"request_digest"`
	PublicationID    string                     `json:"publication_id"`
	RetryOf          string                     `json:"retry_of,omitempty"`
}
type answerProgress struct {
	Candidate           *catalog.AnswerCandidate `json:"candidate,omitempty"`
	AnswerID            string                   `json:"answer_id,omitempty"`
	PackageID           string                   `json:"package_id,omitempty"`
	Accepted            bool                     `json:"accepted"`
	EvidenceCount       int                      `json:"evidence_count"`
	IndependentFamilies int                      `json:"independent_families"`
	Limitations         []string                 `json:"limitations"`
	Provider            answers.Provider         `json:"provider"`
	Route               string                   `json:"route"`
	Budget              answers.Budget           `json:"budget"`
	ProviderTimeoutMS   int                      `json:"provider_timeout_ms"`
	MaxRepairs          int                      `json:"max_repairs"`
}

func (a *App) requestAnswer(ctx context.Context, req contracts.Request) (catalog.Work, error) {
	var empty catalog.Work
	var in AnswerWorkflowRequest
	if contracts.DecodeExplore(req.Data, &in) != nil || !contracts.ValidID(req.RequestID) {
		return empty, contracts.Fail("invalid_request")
	}
	digest := digestJSON(in)
	if prior, e := a.Catalog.Work(ctx, req.RequestID); e == nil {
		var p answerPayload
		if prior.Kind != "answers.request" || strictPayload(prior.Payload, &p) != nil || p.RequestDigest != digest {
			return empty, contracts.Fail("conflict")
		}
		return prior, nil
	} else if contracts.SafeError(e).Code != "not_found" {
		return empty, e
	}
	if in.Configuration != nil && in.PipelineID != "" || in.PipelineRevision < 0 || in.PipelineRevision != 0 && in.PipelineID == "" {
		return empty, contracts.Fail("invalid_request")
	}
	p := answerPayload{Request: in.Request, PipelineID: in.PipelineID, RequestDigest: digest, PublicationID: contracts.ID()}
	if in.Configuration != nil {
		p.Configuration = *in.Configuration
	} else if in.PipelineID != "" {
		if !contracts.ValidID(in.PipelineID) {
			return empty, contracts.Fail("invalid_request")
		}
		saved, e := a.Catalog.Pipeline(ctx, in.PipelineID)
		if e != nil {
			return empty, e
		}
		if in.PipelineRevision != 0 && in.PipelineRevision != saved.Revision {
			return empty, contracts.Fail("conflict")
		}
		d, e := a.configuredPipelineDefinition(saved)
		if e != nil {
			return empty, e
		}
		if d.AnswerSynthesis == nil {
			return empty, contracts.Fail("unsupported_capability")
		}
		p.Configuration = *d.AnswerSynthesis
		p.PipelineRevision = saved.Revision
	} else {
		c, _, e := a.assistanceConfiguration(ctx)
		if e != nil {
			return empty, e
		}
		p.Configuration = assistance.SynthesisConfig{Provider: c}
	}
	var err error
	p.Configuration, err = assistance.NormalizeSynthesis(p.Configuration)
	if err != nil {
		return empty, err
	}
	if p.Request.Provider.ID != "" || p.Request.Provider.Model != "" || p.Request.Provider.ConfigurationDigest != "" || p.Request.Provider.PromptContract != "" {
		return empty, contracts.Fail("invalid_request")
	}
	p.Request.Provider = synthesisProvider(p.Configuration)
	p.Request, err = answers.Normalize(p.Request)
	if err != nil {
		return empty, err
	}
	raw, _ := json.Marshal(p)
	work, err := a.Catalog.EnqueueWork(ctx, req.RequestID, "answers.request", raw)
	if err == nil {
		a.recoverWork()
	}
	return work, err
}
func (a *App) retryAnswer(ctx context.Context, requestID, id string) (catalog.Work, error) {
	var empty catalog.Work
	if !contracts.ValidID(requestID) || !contracts.ValidID(id) || requestID == id {
		return empty, contracts.Fail("invalid_request")
	}
	if prior, e := a.Catalog.Work(ctx, requestID); e == nil {
		var p answerPayload
		if prior.Kind != "answers.request" || strictPayload(prior.Payload, &p) != nil || p.RetryOf != id {
			return empty, contracts.Fail("conflict")
		}
		return prior, nil
	} else if contracts.SafeError(e).Code != "not_found" {
		return empty, e
	}
	original, e := a.Catalog.Work(ctx, id)
	if e != nil {
		return empty, e
	}
	if original.Kind != "answers.request" || original.State != "failed" && original.State != "cancelled" && original.State != "interrupted" {
		return empty, contracts.Fail("conflict")
	}
	var p answerPayload
	if strictPayload(original.Payload, &p) != nil {
		return empty, contracts.Fail("conflict")
	}
	// Immutable election persists; a fresh request selects a different route.
	p.RetryOf = id
	p.PublicationID = contracts.ID()
	p.RequestDigest = digestJSON([]string{"answer-retry", id})
	raw, _ := json.Marshal(p)
	work, e := a.Catalog.EnqueueWork(ctx, requestID, "answers.request", raw)
	if e == nil {
		a.recoverWork()
	}
	return work, e
}
func answerProgressFor(p answerPayload) answerProgress {
	return answerProgress{Provider: synthesisProvider(p.Configuration), Route: p.Configuration.Provider.Route, Budget: p.Request.Budget, ProviderTimeoutMS: p.Configuration.Provider.Limits.TimeoutMS, MaxRepairs: *p.Configuration.MaxRepairs, Limitations: []string{}}
}
func (a *App) processAnswer(parent context.Context, claim catalog.Work) (out answerProgress, err error) {
	ctx, cancel := context.WithTimeout(parent, 180*time.Second)
	defer cancel()
	var p answerPayload
	if strictPayload(claim.Payload, &p) != nil || !contracts.ValidID(p.PublicationID) {
		return out, contracts.Fail("invalid_request")
	}
	normalized, e := assistance.NormalizeSynthesis(p.Configuration)
	if e != nil || !reflect.DeepEqual(normalized, p.Configuration) {
		return out, contracts.Fail("invalid_request")
	}
	request, e := answers.Normalize(p.Request)
	if e != nil || !reflect.DeepEqual(request, p.Request) || !reflect.DeepEqual(p.Request.Provider, synthesisProvider(p.Configuration)) {
		return out, contracts.Fail("invalid_request")
	}
	out = answerProgressFor(p)
	checkpoint := func(phase string) error {
		raw, _ := json.Marshal(out)
		_, e := a.Catalog.CheckpointWork(ctx, claim, phase, "running", raw, leaseTTL)
		return e
	}
	// Candidate metadata precedes publication. Recovery must reconcile it before
	// replacing phases or invoking a nondeterministic provider again.
	var prior answerProgress
	if json.Unmarshal(claim.Result, &prior) == nil && prior.Candidate != nil {
		out = prior
		if !out.Candidate.Valid() || out.Candidate.PublicationID != p.PublicationID {
			return out, contracts.Fail("conflict")
		}
		result, e := a.readGroundedAnswer(ctx, p.PublicationID, out.Candidate.Digest)
		if e != nil {
			return out, e
		}
		if result.WorkID != claim.ID || result.PackageID != out.Candidate.PackageID || result.SourceEpoch != out.Candidate.SourceEpoch {
			return out, contracts.Fail("conflict")
		}
		if e = a.validateAnswer(ctx, result.Answer); e != nil {
			return out, e
		}
		if e = a.Catalog.ArtifactReference(ctx, p.PublicationID, claim.ID, false); e != nil {
			return out, e
		}
		return a.acceptAnswerProgress(ctx, claim, out)
	}
	if err = checkpoint("retrieval"); err != nil {
		return out, err
	}
	prepared, e := a.prepareAnswer(ctx, p.Request)
	if e != nil {
		return out, e
	}
	if prepared.Freshness != "current" {
		return out, contracts.Condition("conflict", "source_authority_changed")
	}
	pkg := prepared.Package
	out.PackageID = pkg.ID
	out.EvidenceCount = len(pkg.Items)
	out.IndependentFamilies = pkg.IndependentFamilies
	out.Limitations = append([]string{}, pkg.Diagnostics...)
	if err = checkpoint("context"); err != nil {
		return out, err
	}
	grounded, e := a.synthesizeAnswer(ctx, pkg, p.Configuration, checkpoint)
	if e != nil {
		return out, e
	}
	grounded.ID = p.PublicationID
	grounded.WorkID = claim.ID
	raw, e := json.Marshal(grounded)
	if e != nil || len(raw) > answers.MaxPackageBytes {
		return out, contracts.Fail("output_limit")
	}
	out.Candidate = &catalog.AnswerCandidate{PublicationID: p.PublicationID, PackageID: pkg.ID, Digest: documentHash(raw), Size: int64(len(raw)), SourceEpoch: pkg.Observation.SourceEpoch, MediaIDs: p.Request.Filters.MediaIDs}
	out.AnswerID = p.PublicationID
	out.Limitations = append(out.Limitations, grounded.Diagnostics...)
	if err = checkpoint("publication"); err != nil {
		return out, err
	}
	if err = a.saveGroundedAnswer(ctx, claim.ID, grounded.ID, raw); err != nil {
		// A publication response can be lost after the exact bytes became available.
		reconciled, readErr := a.readGroundedAnswer(ctx, grounded.ID, out.Candidate.Digest)
		if readErr != nil || reconciled.WorkID != claim.ID {
			return out, err
		}
		if err = a.Catalog.ArtifactReference(ctx, grounded.ID, claim.ID, false); err != nil {
			return out, err
		}
	}
	return a.acceptAnswerProgress(ctx, claim, out)
}
func (a *App) acceptAnswerProgress(ctx context.Context, claim catalog.Work, out answerProgress) (answerProgress, error) {
	out.Accepted = true
	raw, _ := json.Marshal(out)
	_, err := a.Catalog.AcceptAnswer(ctx, claim, *out.Candidate, raw)
	if err != nil {
		actual, e := a.Catalog.Work(ctx, claim.ID)
		if e == nil && actual.Kind == "answers.request" && actual.State == "succeeded" && reflect.DeepEqual(actual.Result, raw) {
			return out, nil
		}
		out.Accepted = false
		return out, err
	}
	return out, nil
}
