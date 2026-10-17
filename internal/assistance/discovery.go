// SPDX-License-Identifier: Apache-2.0
package assistance

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/packageenv"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DiscoveryInput struct {
	Endpoints []string `json:"endpoints,omitempty"`
	TimeoutMS int      `json:"timeout_ms,omitempty"`
}
type Candidate struct {
	Endpoint   string      `json:"endpoint"`
	Sources    []string    `json:"sources"`
	CheckedAt  string      `json:"checked_at"`
	Installed  bool        `json:"installed"`
	Running    bool        `json:"running"`
	Reachable  bool        `json:"reachable"`
	Configured bool        `json:"configured"`
	Tested     bool        `json:"tested"`
	ProbeState string      `json:"probe_state"`
	Info       *OllamaInfo `json:"metadata,omitempty"`
	Correction string      `json:"correction,omitempty"`
}
type Discovery struct {
	Version         string      `json:"schema_version"`
	CheckedAt       string      `json:"checked_at"`
	Candidates      []Candidate `json:"candidates"`
	Installations   []string    `json:"installations"`
	PackageDeclared bool        `json:"package_resource_declared"`
	Truncated       bool        `json:"truncated"`
}

func normalizeHint(h string) (string, error) {
	if !strings.Contains(h, "://") {
		h = "http://" + h
	}
	u, e := url.Parse(h)
	if e != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return "", contracts.Fail("invalid_request")
	}
	host := u.Hostname()
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	if host == "::" {
		host = "::1"
	}
	port := u.Port()
	if port == "" {
		port = "11434"
	}
	u.Host = net.JoinHostPort(host, port)
	u.Path = ""
	if !validEndpoint(u.String()) {
		return "", contracts.Fail("invalid_request")
	}
	return u.String(), nil
}
func (a Ollama) Discover(ctx context.Context, in DiscoveryInput, c Config) (Discovery, error) {
	out := Discovery{Version: contracts.Version, CheckedAt: time.Now().UTC().Format(time.RFC3339Nano), Candidates: []Candidate{}, Installations: []string{}}
	if len(in.Endpoints) > 16 || in.TimeoutMS < 0 || in.TimeoutMS > 30000 {
		return out, contracts.Fail("input_limit")
	}
	if in.TimeoutMS == 0 {
		in.TimeoutMS = 5000
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(in.TimeoutMS)*time.Millisecond)
	defer cancel()
	indices := map[string]int{}
	elect := map[string]bool{}
	add := func(endpoint, source string, probe bool) {
		if !validEndpoint(endpoint) {
			return
		}
		endpoint = strings.TrimRight(endpoint, "/")
		if i, ok := indices[endpoint]; ok {
			out.Candidates[i].Sources = append(out.Candidates[i].Sources, source)
			elect[endpoint] = elect[endpoint] || probe
			return
		}
		if len(out.Candidates) >= 16 {
			out.Truncated = true
			return
		}
		indices[endpoint] = len(out.Candidates)
		elect[endpoint] = probe
		out.Candidates = append(out.Candidates, Candidate{Endpoint: endpoint, Sources: []string{source}, CheckedAt: out.CheckedAt, Configured: strings.TrimRight(c.Endpoint, "/") == endpoint, ProbeState: "untested"})
	}
	resources, resourceErr := packageenv.ServiceResources(ctx)
	if resourceErr == nil {
		for _, resource := range resources {
			if !validEndpoint(resource.Endpoint) {
				continue
			}
			out.PackageDeclared = true
			add(resource.Endpoint, "package-declaration", endpointScope(resource.Endpoint) == "loopback")
			if resource.Executable != "" {
				out.Installations = append(out.Installations, resource.Executable)
			}
		}
	}
	if c.Adapter == "ollama" && c.Endpoint != "" {
		add(c.Endpoint, "configured", true)
	}
	add("http://127.0.0.1:11434", "loopback-default", true)
	add("http://[::1]:11434", "loopback-default", true)
	if hint := os.Getenv("OLLAMA_HOST"); hint != "" {
		normalized, e := normalizeHint(hint)
		if e == nil {
			add(normalized, "OLLAMA_HOST", endpointScope(normalized) == "loopback")
		} else {
			if len(out.Candidates) < 16 {
				out.Candidates = append(out.Candidates, Candidate{Sources: []string{"OLLAMA_HOST"}, CheckedAt: out.CheckedAt, ProbeState: "invalid-hint", Correction: "OLLAMA_HOST is not a credential-free HTTP(S) service endpoint."})
			} else {
				out.Truncated = true
			}
		}
	}
	for _, endpoint := range in.Endpoints {
		if !validEndpoint(endpoint) {
			return out, contracts.Condition("invalid_request", "ollama_endpoint")
		}
		add(endpoint, "manual-election", true)
	}
	name := "ollama"
	if os.PathSeparator == '\\' {
		name += ".exe"
	}
	paths := []string{filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Ollama", name), "/usr/local/bin/ollama", "/usr/bin/ollama"}
	if executable, e := os.Executable(); e == nil {
		paths = append(paths, filepath.Join(filepath.Dir(executable), name))
	}
	dirs := filepath.SplitList(os.Getenv("PATH"))
	if len(dirs) > 128 {
		dirs = dirs[:128]
		out.Truncated = true
	}
	for _, d := range dirs {
		if d != "" {
			paths = append(paths, filepath.Join(d, name))
		}
	}
	seen := map[string]bool{}
	for _, p := range paths {
		if !filepath.IsAbs(p) || seen[p] {
			continue
		}
		seen[p] = true
		if st, e := os.Stat(p); e == nil && st.Mode().IsRegular() {
			out.Installations = append(out.Installations, p)
		}
	}
	for i := range out.Candidates {
		candidate := &out.Candidates[i]
		candidate.Installed = endpointScope(candidate.Endpoint) == "loopback" && len(out.Installations) > 0
		if !elect[candidate.Endpoint] {
			continue
		}
		id := ""
		selected := ""
		if candidate.Configured {
			id = c.CredentialID
			selected = c.Model
		}
		info, e := a.Inspect(ctx, candidate.Endpoint, id, selected)
		if info.Version != "" {
			candidate.Running = true
			candidate.Reachable = true
			candidate.Info = &info
		}
		if e != nil {
			candidate.ProbeState = "unavailable"
			if candidate.Reachable {
				candidate.ProbeState = "metadata-failed"
			}
			candidate.Correction = contracts.SafeError(e).Message
			if err := ProviderContextError(ctx); err != nil {
				return out, err
			}
			continue
		}
		candidate.Running = true
		candidate.Reachable = true
		candidate.ProbeState = "observed"
		candidate.Info = &info
	}
	if err := ProviderContextError(ctx); err != nil {
		return out, err
	}
	return out, nil
}
