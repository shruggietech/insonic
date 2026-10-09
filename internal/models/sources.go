// SPDX-License-Identifier: Apache-2.0
package models

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"net/http"
	"strings"
)

type CatalogEntry struct {
	Selector string   `json:"selector"`
	Manifest Manifest `json:"manifest"`
}
type SourceCatalog struct {
	Kind    string         `json:"kind"`
	Version string         `json:"schema_version"`
	Entries []CatalogEntry `json:"entries"`
}
type Discovery struct {
	Selector    string   `json:"selector"`
	Reference   string   `json:"reference"`
	Manifest    Manifest `json:"manifest"`
	Digest      string   `json:"manifest_digest"`
	State       string   `json:"state"`
	Compatible  bool     `json:"compatible"`
	Diagnostics []string `json:"diagnostics"`
}

func digestBytes(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func (r *Resolver) sourceCatalog(ctx context.Context, ref string) (catalog.ModelSource, SourceCatalog, error) {
	source, e := r.Catalog.ModelSource(ctx, ref)
	var out SourceCatalog
	if e != nil {
		return source, out, e
	}
	if source.State != "active" {
		return source, out, contracts.Fail("not_found")
	}
	if _, e = sourceURL(source.URL, source.LocalHTTP); e != nil {
		return source, out, e
	}
	req, e := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if e != nil {
		return source, out, contracts.Fail("invalid_request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	if source.CredentialID != "" {
		if r.Secrets == nil {
			return source, out, contracts.Fail("unavailable")
		}
		raw, e := r.Secrets.Resolve(ctx, source.CredentialID)
		if e != nil {
			return source, out, contracts.Fail("unavailable")
		}
		token := string(raw)
		if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
			var auth struct {
				Token string `json:"token"`
			}
			if catalog.ValidateJSON(raw) != nil || json.Unmarshal(raw, &auth) != nil {
				clear(raw)
				return source, out, contracts.Fail("invalid_request")
			}
			token = auth.Token
		}
		clear(raw)
		if token == "" || strings.ContainsAny(token, "\r\n") {
			return source, out, contracts.Fail("invalid_request")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, e := r.Client.Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return source, out, contracts.Fail("cancelled")
		}
		return source, out, contracts.Fail("unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Encoding") != "" && response.Header.Get("Content-Encoding") != "identity" {
		return source, out, contracts.Fail("unavailable")
	}
	raw, e := io.ReadAll(io.LimitReader(response.Body, (4<<20)+1))
	if e != nil || len(raw) > 4<<20 {
		return source, out, contracts.Fail("input_limit")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if catalog.ValidateJSON(raw) != nil || d.Decode(&out) != nil || d.Decode(new(any)) != io.EOF || out.Kind != "model-catalog" || out.Version != contracts.Version || len(out.Entries) > 128 {
		return source, out, contracts.Fail("invalid_request")
	}
	seen := map[string]bool{}
	for _, entry := range out.Entries {
		if !modelText(entry.Selector, 256) || seen[entry.Selector] || entry.Manifest.Validate() != nil {
			return source, out, contracts.Fail("invalid_request")
		}
		seen[entry.Selector] = true
	}
	return source, out, nil
}
func (r *Resolver) Discover(ctx context.Context, ref string) ([]Discovery, error) {
	source, doc, e := r.sourceCatalog(ctx, ref)
	if e != nil {
		return nil, e
	}
	out := []Discovery{}
	for _, entry := range doc.Entries {
		res, e := ResolveManifest(entry.Manifest, "")
		if e != nil {
			return nil, e
		}
		res.Compatible = false
		res.Diagnostics = []string{"No compatible implemented consumer is declared for this bundle. Inspect an explicit operation for its requirements."}
		// Discovery answers whether any declared operation has an implemented
		// compatible consumer. Election still inspects exactly its requested task.
		for _, op := range entry.Manifest.Capabilities {
			if !catalog.ModelOperation(op) {
				continue
			}
			candidate, e := ResolveManifest(entry.Manifest, op)
			if e != nil {
				return nil, e
			}
			if candidate.Compatible {
				res = candidate
				break
			}
		}
		if install, e := r.Catalog.BaseModel(ctx, res.Target.ID); e == nil {
			if install.Digest != res.Digest {
				return nil, contracts.Fail("conflict")
			}
			res.State = install.State
		}
		out = append(out, Discovery{Selector: entry.Selector, Reference: "source:" + source.Name + "/" + entry.Selector, Manifest: entry.Manifest, Digest: res.Digest, State: res.State, Compatible: res.Compatible, Diagnostics: res.Diagnostics})
	}
	return out, nil
}
func (r *Resolver) ResolveSource(ctx context.Context, sourceRef, selector, operation string) (Resolution, error) {
	source, doc, e := r.sourceCatalog(ctx, sourceRef)
	if e != nil {
		return Resolution{}, e
	}
	for _, entry := range doc.Entries {
		if entry.Selector != selector {
			continue
		}
		out, e := ResolveManifest(entry.Manifest, operation)
		if e != nil {
			return out, e
		}
		out.Reference = "source:" + source.Name + "/" + selector
		out.SourceID, out.SourceRevision = source.ID, source.Revision
		if install, e := r.Catalog.BaseModel(ctx, out.Target.ID); e == nil {
			if install.Digest != out.Digest {
				return out, contracts.Fail("conflict")
			}
			out.State = install.State
		}
		return out, nil
	}
	return Resolution{}, contracts.Fail("not_found")
}
