// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type ArcadeConfig struct {
	Endpoint         string `json:"endpoint"`
	Database         string `json:"database"`
	CredentialID     string `json:"credential_id,omitempty"`
	PreferredDialect string `json:"preferred_dialect,omitempty"`
}
type arcadeEngine struct {
	config   ArcadeConfig
	provider contracts.SecretProvider
	client   *http.Client
}
type arcadeSession struct {
	engine   *arcadeEngine
	id       string
	ended    bool
	readonly bool
}

var databaseName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

func OpenArcade(c ArcadeConfig, p contracts.SecretProvider) (Adapter, error) {
	u, e := url.Parse(c.Endpoint)
	if e != nil || !u.IsAbs() || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") || !databaseName.MatchString(c.Database) {
		return nil, contracts.Fail("invalid_request")
	}
	if c.CredentialID != "" {
		if !contracts.ValidID(c.CredentialID) || p == nil {
			return nil, contracts.Fail("invalid_request")
		}
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "https" && !strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback()) {
			return nil, contracts.Fail("invalid_request")
		}
	}
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	e2 := &arcadeEngine{config: c, provider: p, client: client}
	return &adapter{engine: e2, backend: "arcadedb", version: e2.version(context.Background())}, nil
}
func (e *arcadeEngine) Close() { e.client.CloseIdleConnections() }
func (e *arcadeEngine) call(ctx context.Context, route string, body any, sessionID string) (Rows, string, error) {
	raw, err := json.Marshal(body)
	if err != nil || len(raw) > 8<<20 {
		return nil, "", contracts.Fail("input_limit")
	}
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(e.config.Endpoint, "/")+"/api/v1/"+route+"/"+url.PathEscape(e.config.Database), bytes.NewReader(raw))
	if err != nil {
		return nil, "", contracts.Fail("invalid_request")
	}
	req.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		req.Header.Set("arcadedb-session-id", sessionID)
	}
	if err = e.authorize(ctx, req); err != nil {
		return nil, "", err
	}
	res, err := e.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, "", contracts.Fail("cancelled")
		}
		return nil, "", contracts.Fail("unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, "", contracts.Fail("operation_failed")
	}
	sid := res.Header.Get("arcadedb-session-id")
	if res.StatusCode == 204 {
		return Rows{}, sid, nil
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, (8<<20)+1))
	if err != nil || len(data) > 8<<20 {
		return nil, "", contracts.Fail("output_limit")
	}
	var result struct {
		Result    Rows `json:"result"`
		Truncated bool `json:"truncated"`
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if d.Decode(&result) != nil || d.Decode(new(any)) != io.EOF {
		return nil, "", contracts.Fail("operation_failed")
	}
	if result.Truncated || len(result.Result) > 200000 {
		return nil, "", contracts.Fail("output_limit")
	}
	return result.Result, sid, nil
}
func (e *arcadeEngine) Begin(ctx context.Context, readonly bool) (session, error) {
	_, id, err := e.call(ctx, "begin", map[string]any{}, "")
	if err != nil {
		return nil, err
	}
	if id == "" {
		return nil, contracts.Fail("operation_failed")
	}
	return &arcadeSession{engine: e, id: id, readonly: readonly}, nil
}
func (s *arcadeSession) Query(ctx context.Context, write bool, q string, p map[string]any) (Rows, error) {
	if s.readonly && write {
		return nil, contracts.Fail("unsupported_capability")
	}
	route := "query"
	if write {
		route = "command"
	}
	rows, _, e := s.engine.call(ctx, route, map[string]any{"language": "sql", "command": q, "params": p, "limit": 200001}, s.id)
	return rows, e
}
func (s *arcadeSession) Native(ctx context.Context, dialect, q string, p map[string]any) (Rows, error) {
	language := "sql"
	if dialect == "arcade-opencypher" {
		language = "opencypher"
	}
	rows, _, e := s.engine.call(ctx, "query", map[string]any{"language": language, "command": q, "params": p, "limit": 501}, s.id)
	if len(rows) > 500 {
		return nil, contracts.Fail("output_limit")
	}
	return rows, e
}
func (s *arcadeSession) Commit(ctx context.Context) error {
	_, _, e := s.engine.call(ctx, "commit", map[string]any{}, s.id)
	if e == nil {
		s.ended = true
	}
	return e
}
func (s *arcadeSession) Rollback(ctx context.Context) {
	if s.ended {
		return
	}
	if ctx.Err() != nil {
		ctx = context.Background()
	}
	ctx, stop := context.WithTimeout(ctx, 3*time.Second)
	defer stop()
	s.engine.call(ctx, "rollback", map[string]any{}, s.id)
	s.ended = true
}

func (e *arcadeEngine) authorize(ctx context.Context, req *http.Request) error {
	if e.config.CredentialID != "" {
		raw, err := e.provider.Resolve(ctx, e.config.CredentialID)
		if err != nil {
			return contracts.Fail("unavailable")
		}
		defer clear(raw)
		var auth struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if contracts.DecodeExplore(raw, &auth) != nil || auth.Username == "" {
			return contracts.Fail("invalid_request")
		}
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	return nil
}

func (e *arcadeEngine) version(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(e.config.Endpoint, "/")+"/api/v1/server", nil)
	if err != nil || e.authorize(ctx, req) != nil {
		return "unknown"
	}
	res, err := e.client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "unknown"
	}
	var info struct {
		Version string `json:"version"`
	}
	if json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&info) != nil || len(info.Version) > 512 {
		return "unknown"
	}
	fields := strings.Fields(info.Version)
	if len(fields) == 0 {
		return "unknown"
	}
	return fields[0]
}
