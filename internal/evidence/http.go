// SPDX-License-Identifier: Apache-2.0
package evidence

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func ValidateHTTP(c Config) error {
	u, e := url.Parse(c.Endpoint)
	if e != nil || !u.IsAbs() || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "http" && u.Scheme != "https") {
		return contracts.Fail("invalid_request")
	}
	if c.CredentialID != "" {
		if !contracts.ValidID(c.CredentialID) {
			return contracts.Fail("invalid_request")
		}
		ip := net.ParseIP(u.Hostname())
		if u.Scheme != "https" && !strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback()) {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}

type HTTPAdapter struct{ Secrets contracts.SecretProvider }

func (a HTTPAdapter) Extract(ctx context.Context, ch Chunk, c Config) ([]Assertion, error) {
	if e := ValidateHTTP(c); e != nil {
		return nil, e
	}
	ctx, stop := context.WithTimeout(ctx, time.Duration(c.TimeoutMS)*time.Millisecond)
	defer stop()
	raw, _ := json.Marshal(map[string]any{"contract_version": "1", "operation": "assertion-extraction", "model": c.Model, "chunk": ch})
	req, e := http.NewRequestWithContext(ctx, "POST", c.Endpoint, bytes.NewReader(raw))
	if e != nil {
		return nil, contracts.Fail("invalid_request")
	}
	req.Header.Set("Content-Type", "application/json")
	if c.CredentialID != "" {
		if a.Secrets == nil {
			return nil, contracts.Fail("unavailable")
		}
		raw, e := a.Secrets.Resolve(ctx, c.CredentialID)
		if e != nil {
			return nil, e
		}
		defer clear(raw)
		req.Header.Set("Authorization", "Bearer "+string(raw))
	}
	client := &http.Client{Timeout: time.Duration(c.TimeoutMS) * time.Millisecond, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	res, e := client.Do(req)
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, contracts.Fail("operation_failed")
	}
	raw, e = io.ReadAll(io.LimitReader(res.Body, int64(c.MaxResponseBytes)+1))
	if e != nil || len(raw) > c.MaxResponseBytes {
		return nil, contracts.Fail("output_limit")
	}
	var result struct {
		ContractVersion string      `json:"contract_version"`
		Assertions      []Assertion `json:"assertions"`
	}
	if contracts.DecodeExplore(raw, &result) != nil || result.ContractVersion != "1" {
		return nil, contracts.Fail("invalid_request")
	}
	return result.Assertions, nil
}
