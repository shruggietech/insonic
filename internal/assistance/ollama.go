// SPDX-License-Identifier: Apache-2.0
package assistance

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Ollama struct {
	Secrets contracts.SecretProvider
	Client  *http.Client
}
type OllamaModel struct {
	Name           string   `json:"name"`
	Digest         string   `json:"digest"`
	Size           int64    `json:"size"`
	Capabilities   []string `json:"capabilities"`
	InferenceRoute string   `json:"inference_route"`
	Compatible     bool     `json:"task_compatible"`
	RemoteModel    string   `json:"remote_model,omitempty"`
	RemoteHost     string   `json:"remote_host,omitempty"`
	DetailState    string   `json:"detail_state"`
}
type OllamaInfo struct {
	Endpoint      string        `json:"endpoint"`
	EndpointScope string        `json:"endpoint_scope"`
	Version       string        `json:"version"`
	Models        []OllamaModel `json:"models"`
	Truncated     bool          `json:"truncated"`
	CheckedAt     string        `json:"checked_at"`
}

func endpointScope(endpoint string) string {
	u, e := url.Parse(endpoint)
	if e != nil {
		return "unknown"
	}
	ip := net.ParseIP(u.Hostname())
	if strings.EqualFold(u.Hostname(), "localhost") || ip != nil && ip.IsLoopback() {
		return "loopback"
	}
	return "connected"
}
func validEndpoint(endpoint string) bool {
	u, e := url.Parse(endpoint)
	return e == nil && len(endpoint) <= 2048 && u.IsAbs() && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && (u.Scheme == "http" || u.Scheme == "https") && (u.Path == "" || u.Path == "/")
}
func (a Ollama) call(ctx context.Context, endpoint, route, id string, body any, cap int, out any) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	if !validEndpoint(endpoint) {
		return contracts.Condition("invalid_request", "ollama_endpoint")
	}
	method := "GET"
	var reader io.Reader
	if body != nil {
		method = "POST"
		raw, e := json.Marshal(body)
		if e != nil || len(raw) > 128<<10 {
			return contracts.Fail("input_limit")
		}
		reader = bytes.NewReader(raw)
	}
	req, e := http.NewRequestWithContext(ctx, method, strings.TrimRight(endpoint, "/")+route, reader)
	if e != nil {
		return contracts.Fail("invalid_request")
	}
	req.Header.Set("Content-Type", "application/json")
	if id != "" {
		if a.Secrets == nil {
			return contracts.Condition("unavailable", "credential_bootstrap")
		}
		secret, e := a.Secrets.Resolve(ctx, id)
		if e != nil {
			return contracts.Condition("unavailable", "credential_bootstrap")
		}
		defer clear(secret)
		req.Header.Set("Authorization", "Bearer "+string(secret))
	}
	client := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if a.Client != nil {
		client.Transport = a.Client.Transport
	}
	defer client.CloseIdleConnections()
	res, e := client.Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return contracts.Fail("cancelled")
		}
		return contracts.Condition("unavailable", "ollama_unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		reason := "ollama_provider"
		switch res.StatusCode {
		case 401, 403:
			reason = "ollama_auth"
		case 404:
			reason = "ollama_missing"
		case 429, 503:
			reason = "ollama_busy"
		case 301, 302, 303, 307, 308:
			reason = "ollama_redirect"
		}
		return contracts.Condition("unavailable", reason)
	}
	raw, e := io.ReadAll(io.LimitReader(res.Body, int64(cap)+1))
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	if e != nil || len(raw) > cap {
		return contracts.Fail("output_limit")
	}
	if catalog.ValidateJSON(raw) != nil || json.Unmarshal(raw, out) != nil {
		return contracts.Condition("operation_failed", "ollama_protocol")
	}
	return nil
}
func (a Ollama) model(ctx context.Context, endpoint, id, name string) (OllamaModel, error) {
	m := OllamaModel{Name: name, InferenceRoute: "unknown", DetailState: "untested"}
	var meta struct {
		Capabilities []string `json:"capabilities"`
		RemoteModel  string   `json:"remote_model"`
		RemoteHost   string   `json:"remote_host"`
		Details      struct {
			Format string `json:"format"`
		} `json:"details"`
	}
	if strings.TrimSpace(name) == "" || len(name) > 512 {
		return m, contracts.Fail("invalid_request")
	}
	if e := a.call(ctx, endpoint, "/api/show", id, map[string]any{"model": name, "verbose": false}, 1<<20, &meta); e != nil {
		return m, e
	}
	m.DetailState = "observed"
	if len(meta.Capabilities) > 32 {
		return m, contracts.Fail("output_limit")
	}
	for _, c := range meta.Capabilities {
		if len(c) > 128 {
			return m, contracts.Fail("output_limit")
		}
		if c == "completion" {
			m.Compatible = true
		}
	}
	m.Capabilities = meta.Capabilities
	// Return endpoint classes rather than arbitrary remote URL values that might contain secrets.
	if meta.RemoteModel != "" || meta.RemoteHost != "" {
		m.InferenceRoute = "hosted"
		m.RemoteModel = "advertised"
		m.RemoteHost = "advertised"
	} else if meta.Details.Format != "" {
		m.InferenceRoute = "local"
	}
	return m, nil
}
func (a Ollama) Inspect(ctx context.Context, endpoint, id, selected string) (OllamaInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out := OllamaInfo{Endpoint: endpoint, EndpointScope: endpointScope(endpoint), Models: []OllamaModel{}, CheckedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	var version struct {
		Version string `json:"version"`
	}
	if e := a.call(ctx, endpoint, "/api/version", id, nil, 1<<20, &version); e != nil {
		return out, e
	}
	if version.Version == "" || len(version.Version) > 128 {
		return out, contracts.Condition("operation_failed", "ollama_protocol")
	}
	out.Version = version.Version
	var tags struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
			Size   int64  `json:"size"`
		} `json:"models"`
	}
	if e := a.call(ctx, endpoint, "/api/tags", id, nil, 1<<20, &tags); e != nil {
		return out, e
	}
	if len(tags.Models) > 128 {
		tags.Models = tags.Models[:128]
		out.Truncated = true
	}
	details := 0
	for _, tag := range tags.Models {
		if tag.Name == "" || len(tag.Name) > 512 || len(tag.Digest) > 512 || tag.Size < 0 {
			return out, contracts.Condition("operation_failed", "ollama_protocol")
		}
		m := OllamaModel{Name: tag.Name, Digest: tag.Digest, Size: tag.Size, InferenceRoute: "unknown", DetailState: "untested"}
		if details < 8 || tag.Name == selected {
			var e error
			m, e = a.model(ctx, endpoint, id, tag.Name)
			if e != nil {
				if ctx.Err() != nil {
					return out, e
				}
				m.DetailState = "failed"
			}
			m.Digest = tag.Digest
			m.Size = tag.Size
			details++
		} else {
			out.Truncated = true
		}
		out.Models = append(out.Models, m)
	}
	if ctx.Err() != nil {
		return out, contracts.Fail("cancelled")
	}
	return out, nil
}
func (a Ollama) Propose(ctx context.Context, input Request, c Config) (Proposal, error) {
	var empty Proposal
	if a.Client == nil && InCI() {
		return empty, contracts.Fail("engine_ci_forbidden")
	}
	var e error
	c, e = Normalize(c)
	if e != nil {
		return empty, e
	}
	if !c.Enabled {
		return empty, contracts.Fail("unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.Limits.TimeoutMS)*time.Millisecond)
	defer cancel()
	m, e := a.model(ctx, c.Endpoint, c.CredentialID, c.Model)
	if e != nil {
		return empty, e
	}
	if !m.Compatible {
		return empty, contracts.Condition("unsupported_capability", "ollama_capability")
	}
	if c.Route == "local" && m.InferenceRoute != "local" {
		return empty, contracts.Condition("invalid_request", "ollama_route")
	}
	raw, e := json.Marshal(input)
	if e != nil || len(raw) > 128<<10 {
		return empty, contracts.Fail("input_limit")
	}
	body := map[string]any{"model": c.Model, "stream": false, "messages": []map[string]string{{"role": "system", "content": "Return exactly one JSON object with contract_version equal to 1, query containing only title, definition and parameters from the supplied query_contract. Omit kind, schema_version, workspace_id, query_id, revision and access_mode from query. Include a bounded pagination limit for normalized queries. Return and a nonempty explanation. Choose a bounded read-only query. Do not return Markdown, tool calls or other text."}, {"role": "user", "content": string(raw)}}, "options": map[string]any{"temperature": 0}}
	if m.InferenceRoute == "local" {
		body["format"] = "json"
	}
	var response struct {
		Done        bool   `json:"done"`
		Error       string `json:"error"`
		RemoteHost  string `json:"remote_host"`
		RemoteModel string `json:"remote_model"`
		Message     struct {
			Role      string            `json:"role"`
			Content   string            `json:"content"`
			ToolCalls []json.RawMessage `json:"tool_calls"`
		} `json:"message"`
	}
	if e = a.call(ctx, c.Endpoint, "/api/chat", c.CredentialID, body, c.Limits.MaxResponseBytes, &response); e != nil {
		return empty, e
	}
	if !response.Done || response.Error != "" || response.Message.Role != "assistant" || len(response.Message.ToolCalls) > 0 {
		return empty, contracts.Condition("operation_failed", "ollama_protocol")
	}
	if c.Route == "local" && (response.RemoteHost != "" || response.RemoteModel != "") {
		return empty, contracts.Condition("invalid_request", "ollama_route")
	}
	if ctx.Err() != nil {
		return empty, contracts.Fail("cancelled")
	}
	proposal, err := Parse([]byte(response.Message.Content))
	proposal.ObservedRoute = m.InferenceRoute
	return proposal, err
}
