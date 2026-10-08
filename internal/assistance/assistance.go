// SPDX-License-Identifier: Apache-2.0
package assistance

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"net"
	"net/url"
	"strings"
)

type Limits struct {
	TimeoutMS        int `json:"timeout_ms"`
	QueryTimeoutMS   int `json:"query_timeout_ms"`
	MaxPromptBytes   int `json:"max_prompt_bytes"`
	MaxResponseBytes int `json:"max_response_bytes"`
	MaxRows          int `json:"max_rows"`
	MaxResultBytes   int `json:"max_result_bytes"`
	MaxContextRows   int `json:"max_context_rows"`
	MaxContextBytes  int `json:"max_context_bytes"`
}
type Config struct {
	Enabled         bool   `json:"enabled"`
	Adapter         string `json:"adapter"`
	ContractVersion string `json:"contract_version"`
	Route           string `json:"route"`
	Endpoint        string `json:"endpoint,omitempty"`
	Model           string `json:"model,omitempty"`
	CredentialID    string `json:"credential_id,omitempty"`
	Mode            string `json:"mode"`
	Limits          Limits `json:"limits"`
}

func DefaultConfig() Config {
	return Config{Adapter: "insonic-http", ContractVersion: "1", Route: "local", Mode: "suggest", Limits: Limits{30000, 20000, 16384, 262144, 100, 524288, 25, 32768}}
}
func Normalize(c Config) (Config, error) {
	d := DefaultConfig()
	if c.Adapter == "" {
		c.Adapter = d.Adapter
	}
	if c.ContractVersion == "" {
		c.ContractVersion = d.ContractVersion
	}
	if c.Route == "" {
		c.Route = d.Route
	}
	if c.Mode == "" {
		c.Mode = d.Mode
	}
	values := []*int{&c.Limits.TimeoutMS, &c.Limits.QueryTimeoutMS, &c.Limits.MaxPromptBytes, &c.Limits.MaxResponseBytes, &c.Limits.MaxRows, &c.Limits.MaxResultBytes, &c.Limits.MaxContextRows, &c.Limits.MaxContextBytes}
	defaults := []int{30000, 20000, 16384, 262144, 100, 524288, 25, 32768}
	caps := []int{60000, 20000, 16384, 524288, 500, 524288, 25, 32768}
	for i, p := range values {
		if *p == 0 {
			*p = defaults[i]
		}
		if *p < 1 || *p > caps[i] {
			return c, contracts.Fail("invalid_request")
		}
	}
	if c.Adapter != "insonic-http" || c.ContractVersion != "1" || (c.Route != "local" && c.Route != "hosted") || (c.Mode != "suggest" && c.Mode != "auto-run") || len(c.Model) > 512 {
		return c, contracts.Fail("invalid_request")
	}
	if c.CredentialID != "" && !contracts.ValidID(c.CredentialID) {
		return c, contracts.Fail("invalid_request")
	}
	if !c.Enabled && c.Endpoint == "" && c.Model == "" {
		return c, nil
	}
	u, e := url.Parse(c.Endpoint)
	if e != nil || !u.IsAbs() || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") || len(c.Endpoint) > 2048 || strings.TrimSpace(c.Model) == "" {
		return c, contracts.Fail("invalid_request")
	}
	ip := net.ParseIP(u.Hostname())
	local := strings.EqualFold(u.Hostname(), "localhost") || (ip != nil && ip.IsLoopback())
	if c.Route == "local" && !local {
		return c, contracts.Fail("invalid_request")
	}
	if c.CredentialID != "" && u.Scheme != "https" && !local {
		return c, contracts.Fail("invalid_request")
	}
	return c, nil
}

type Request struct {
	ContractVersion string          `json:"contract_version"`
	Operation       string          `json:"operation"`
	Model           string          `json:"model"`
	Prompt          string          `json:"prompt"`
	Schema          json.RawMessage `json:"schema"`
	Capabilities    map[string]any  `json:"capabilities"`
	Limits          Limits          `json:"limits"`
	Context         json.RawMessage `json:"context,omitempty"`
}
type Proposal struct {
	ContractVersion string               `json:"contract_version"`
	Query           contracts.QueryInput `json:"query"`
	Explanation     string               `json:"explanation"`
}
type Fixture func(context.Context, Request, Config) (Proposal, error)

func Parse(raw []byte) (Proposal, error) {
	var p Proposal
	if catalog.ValidateJSON(raw) != nil || contracts.DecodeExplore(raw, &p) != nil || p.ContractVersion != "1" || strings.TrimSpace(p.Explanation) == "" || len(p.Explanation) > 8192 || p.Query.Validate() != nil {
		return p, contracts.Fail("invalid_request")
	}
	return p, nil
}
