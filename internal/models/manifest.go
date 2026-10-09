// SPDX-License-Identifier: Apache-2.0
package models

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
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
	if m.Kind != "base-model-manifest" || m.Version != contracts.Version || len(m.Name) == 0 || len(m.Name) > 256 || len(m.ModelVersion) == 0 || len(m.ModelVersion) > 256 || len(m.Revision) == 0 || len(m.Revision) > 512 || len(m.License) == 0 || len(m.Files) == 0 || len(m.Files) > 64 || len(m.Capabilities) == 0 || len(m.Capabilities) > 32 {
		return contracts.Fail("invalid_request")
	}
	roles := map[string]bool{}
	for _, f := range m.Files {
		if f.Role == "" || len(f.Role) > 128 || roles[f.Role] || !digestPattern.MatchString(f.SHA256) || f.Size < 0 || f.CredentialID != "" && !contracts.ValidID(f.CredentialID) {
			return contracts.Fail("invalid_request")
		}
		if _, e := sourceURL(f.URL, f.LocalHTTP); e != nil {
			return e
		}
		roles[f.Role] = true
	}
	for _, c := range m.Capabilities {
		if c == "" || len(c) > 128 {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
func DecodeManifest(raw []byte, out *Manifest) error {
	if len(raw) > 1<<20 || catalog.ValidateJSON(raw) != nil {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return contracts.Fail("invalid_request")
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
