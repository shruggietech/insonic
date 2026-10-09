// SPDX-License-Identifier: Apache-2.0
package models

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"net/http"
	"strings"
	"time"
)

type Resolution struct {
	Reference        string                       `json:"reference"`
	Target           catalog.ModelTarget          `json:"target"`
	Manifest         *Manifest                    `json:"manifest,omitempty"`
	Digest           string                       `json:"manifest_digest"`
	UpstreamRevision string                       `json:"upstream_revision"`
	State            string                       `json:"state"`
	AliasID          string                       `json:"alias_id,omitempty"`
	AliasRevision    int64                        `json:"alias_revision,omitempty"`
	SourceID         string                       `json:"source_id,omitempty"`
	SourceRevision   int64                        `json:"source_revision,omitempty"`
	Compatible       bool                         `json:"compatible"`
	Diagnostics      []string                     `json:"diagnostics"`
	Lineage          *catalog.SpeakerModelLineage `json:"lineage,omitempty"`
}
type Resolver struct {
	Catalog catalog.Catalog
	Client  *http.Client
	Secrets contracts.SecretProvider
}

func NewResolver(c catalog.Catalog, p contracts.SecretProvider) *Resolver {
	return &Resolver{Catalog: c, Secrets: p, Client: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 4 || req.URL.Scheme != via[0].URL.Scheme || req.URL.Host != via[0].URL.Host {
			return contracts.Fail("unavailable")
		}
		_, e := contracts.SourceURL(req.URL.String())
		return e
	}}}
}
func sourceReference(v string) (string, string, bool) {
	if !strings.HasPrefix(v, "source:") {
		return "", "", false
	}
	name, selector, ok := strings.Cut(strings.TrimPrefix(v, "source:"), "/")
	return name, selector, ok && catalog.ValidModelName(name) && modelText(selector, 256)
}
func ValidateReference(v string) bool {
	if contracts.ValidID(v) || catalog.ValidModelName(v) {
		return true
	}
	for _, prefix := range []string{"base:", "speaker:"} {
		if strings.HasPrefix(v, prefix) {
			return contracts.ValidID(strings.TrimPrefix(v, prefix))
		}
	}
	_, _, ok := sourceReference(v)
	return ok
}
func InstallationID(m Manifest) string {
	raw, _ := json.Marshal([]string{m.Name, m.ModelVersion})
	return StableID("model:" + string(raw))
}
func AcquisitionID(workspace string, m Manifest) string {
	return StableID("model-acquisition:" + workspace + ":" + InstallationID(m) + ":" + m.Digest())
}
func ResolveManifest(m Manifest, operation string) (Resolution, error) {
	raw, _ := json.Marshal(m)
	var copy Manifest
	if DecodeManifest(raw, &copy) != nil {
		return Resolution{}, contracts.Fail("invalid_request")
	}
	m = copy
	out := Resolution{Reference: "base:" + InstallationID(m), Target: catalog.ModelTarget{Kind: "base", ID: InstallationID(m), Operation: operation}, Manifest: &m, Digest: m.Digest(), UpstreamRevision: m.Revision, State: "discovered", Diagnostics: []string{}}
	if m.Validate() != nil {
		return out, contracts.Fail("invalid_request")
	}
	if operation == "" {
		out.Compatible = true
		return out, nil
	}
	if !catalog.ModelOperation(operation) {
		return out, contracts.Fail("invalid_request")
	}
	out.Target.Adapter, out.Target.ContractVersion = DefaultAdapter(operation)
	if e := CheckCompatibility(m, operation, out.Target.Adapter, out.Target.ContractVersion); e != nil {
		out.Diagnostics = append(out.Diagnostics, fmt.Sprintf("Selected %s bundle is incompatible with %s contract %s: %v. Supply a compatible bundle or adapter.", operation, out.Target.Adapter, out.Target.ContractVersion, e))
		return out, nil
	}
	out.Compatible = true
	return out, nil
}
func RequireCompatible(r Resolution) error {
	if !r.Compatible {
		return contracts.Fail("unsupported_capability")
	}
	return nil
}
func (r *Resolver) Resolve(ctx context.Context, ref, operation string) (Resolution, error) {
	out := Resolution{Reference: ref, Diagnostics: []string{}}
	if !ValidateReference(ref) || !catalog.ModelOperation(operation) {
		return out, contracts.Fail("invalid_request")
	}
	if name, selector, ok := sourceReference(ref); ok {
		return r.ResolveSource(ctx, name, selector, operation)
	}
	target := catalog.ModelTarget{Kind: "base", ID: strings.TrimPrefix(ref, "base:"), Operation: operation}
	if strings.HasPrefix(ref, "speaker:") {
		target.Kind = "speaker"
		target.ID = strings.TrimPrefix(ref, "speaker:")
	}
	if catalog.ValidModelName(ref) {
		alias, e := r.Catalog.ModelAlias(ctx, ref)
		if e != nil {
			return out, e
		}
		if alias.State != "active" {
			return out, contracts.Fail("not_found")
		}
		target, e = catalog.DecodeModelTarget(alias.Target)
		if e != nil {
			return out, e
		}
		out.AliasID, out.AliasRevision = alias.ID, alias.Revision
		if target.Operation != operation {
			return out, contracts.Fail("unsupported_capability")
		}
	}
	got, e := r.ResolveTarget(ctx, target, operation)
	if e != nil {
		return out, e
	}
	got.Reference = ref
	got.AliasID, got.AliasRevision = out.AliasID, out.AliasRevision
	return got, nil
}
func (r *Resolver) ResolveTarget(ctx context.Context, target catalog.ModelTarget, operation string) (Resolution, error) {
	out := Resolution{Target: target, Diagnostics: []string{}}
	raw, _ := json.Marshal(target)
	if _, e := catalog.DecodeModelTarget(raw); e != nil {
		return out, e
	}
	if target.Operation != operation {
		return out, contracts.Fail("unsupported_capability")
	}
	switch target.Kind {
	case "base":
		install, e := r.Catalog.BaseModel(ctx, target.ID)
		if e != nil {
			return out, e
		}
		var m Manifest
		if DecodeManifest(install.Manifest, &m) != nil || m.Digest() != install.Digest || InstallationID(m) != install.ID {
			return out, contracts.Fail("conflict")
		}
		out, e = ResolveManifest(m, operation)
		if e != nil {
			return out, e
		}
		out.Target = target
		out.State = install.State
		if target.Adapter != "" {
			if e = CheckCompatibility(m, operation, target.Adapter, target.ContractVersion); e != nil {
				out.Compatible = false
				out.Diagnostics = []string{"The selected bundle is incompatible with the elected adapter contract."}
			}
		}
		if out.Target.Adapter == "" {
			out.Target.Adapter, out.Target.ContractVersion = DefaultAdapter(operation)
		}
	case "speaker":
		lineage, e := r.Catalog.SpeakerModelVersion(ctx, target.ID)
		if e != nil {
			return out, e
		}
		out.Lineage = &lineage
		out.State = lineage.Version.State
		if lineage.Version.ManifestArtifactID != nil {
			a, e := r.Catalog.Artifact(ctx, *lineage.Version.ManifestArtifactID)
			if e != nil {
				return out, e
			}
			out.Digest = a.Digest
		}
		out.Diagnostics = []string{"Exact trained version and producing family/run/corpus/artifact lineage are retained. A compatible trained-model consumer is required before execution."}
	case "hosted":
		out.State = "hosted-only"
		out.UpstreamRevision = target.UpstreamRevision
		out.Digest = StableManifestDigest(target)
		out.Compatible = operation == "transcription" || operation == "diarization"
		if !out.Compatible {
			out.Diagnostics = []string{"The configured hosted adapter has no implemented consumer for this operation."}
		}
	}
	return out, nil
}
func StableManifestDigest(value any) string { raw, _ := json.Marshal(value); return digestBytes(raw) }
