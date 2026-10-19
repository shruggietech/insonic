// SPDX-License-Identifier: Apache-2.0
package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"net"
	"net/url"
	"regexp"
)

type File struct {
	Role         string `json:"role"`
	SHA256       string `json:"sha256"`
	Size         int64  `json:"size"`
	URL          string `json:"url"`
	CredentialID string `json:"credential_id,omitempty"`
	LocalHTTP    bool   `json:"local_http,omitempty"`
}
type Manifest struct {
	Kind          string         `json:"kind"`
	Version       string         `json:"schema_version"`
	Name          string         `json:"name"`
	ModelVersion  string         `json:"model_version"`
	Revision      string         `json:"upstream_revision"`
	Capabilities  []string       `json:"capabilities"`
	License       string         `json:"license"`
	Files         []File         `json:"files"`
	Compatibility *Compatibility `json:"compatibility,omitempty"`
}

var digestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func sourceURL(raw string, local bool) (*url.URL, error) {
	u, e := contracts.SourceURL(raw)
	if e != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if u.Scheme != "https" {
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "http" || !local || !(u.Hostname() == "localhost" || ip != nil && ip.IsLoopback()) {
			return nil, contracts.Fail("invalid_request")
		}
	}
	return u, nil
}
func (m Manifest) Validate() error {
	if m.Compatibility != nil && m.Compatibility.Validate() != nil {
		return contracts.Fail("invalid_request")
	}
	if m.Kind != "base-model-manifest" || !contracts.SupportedDocumentVersion(m.Version) || !contracts.ValidTextLength(m.Name, 1, 256) || !contracts.ValidTextLength(m.ModelVersion, 1, 256) || !contracts.ValidTextLength(m.Revision, 1, 512) || !contracts.ValidTextLength(m.License, 1, -1) || len(m.Files) == 0 || len(m.Files) > 64 || len(m.Capabilities) == 0 || len(m.Capabilities) > 32 {
		return contracts.Fail("invalid_request")
	}
	roles := map[string]bool{}
	for _, f := range m.Files {
		if !contracts.ValidTextLength(f.Role, 1, 128) || roles[f.Role] || !digestPattern.MatchString(f.SHA256) || f.Size < 0 || f.CredentialID != "" && !contracts.ValidID(f.CredentialID) {
			return contracts.Fail("invalid_request")
		}
		if _, e := sourceURL(f.URL, f.LocalHTTP); e != nil {
			return e
		}
		roles[f.Role] = true
	}
	for _, c := range m.Capabilities {
		if !contracts.ValidTextLength(c, 1, 128) {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
func DecodeManifest(raw []byte, out *Manifest) error {
	if len(raw) > 1<<20 {
		return contracts.InputFailure("input_bytes", contracts.InputViolation{Constraint: "maxBytes", Bound: 1 << 20})
	}
	if err := schemas.ValidateBaseModelManifest(raw); err != nil {
		return err
	}
	if err := contracts.DecodeInput(raw, out); err != nil {
		return err
	}
	return out.Validate()
}
func (m Manifest) Digest() string {
	raw, _ := json.Marshal(m)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func StableID(value string) string {
	h := sha256.Sum256([]byte(value))
	h[6] = h[6]&15 | 0x50
	h[8] = h[8]&63 | 0x80
	s := hex.EncodeToString(h[:16])
	return s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
}
