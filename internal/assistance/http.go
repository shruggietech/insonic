// SPDX-License-Identifier: Apache-2.0
package assistance

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"net/http"
	"os"
	"time"
)

type HTTPAdapter struct {
	Secrets contracts.SecretProvider
	Client  *http.Client
}

func InCI() bool {
	for _, k := range []string{"CI", "GITHUB_ACTIONS", "TF_BUILD", "BUILD_BUILDID", "JENKINS_URL"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}
func (a HTTPAdapter) Propose(ctx context.Context, input Request, c Config) (Proposal, error) {
	if c.Adapter == "ollama" {
		return (Ollama{Secrets: a.Secrets, Client: a.Client}).Propose(ctx, input, c)
	}
	var empty Proposal
	if a.Client == nil && InCI() {
		return empty, contracts.Fail("engine_ci_forbidden")
	}
	if err := ProviderContextError(ctx); err != nil {
		return empty, err
	}
	var e error
	c, e = Normalize(c)
	if e != nil {
		return empty, e
	}
	if !c.Enabled {
		return empty, contracts.Fail("unavailable")
	}
	bounded, stop := context.WithTimeout(ctx, time.Duration(c.Limits.TimeoutMS)*time.Millisecond)
	defer stop()
	raw, e := json.Marshal(input)
	if e != nil || len(raw) > 128<<10 {
		return empty, contracts.Fail("input_limit")
	}
	req, e := http.NewRequestWithContext(bounded, "POST", c.Endpoint, bytes.NewReader(raw))
	if e != nil {
		return empty, contracts.Fail("invalid_request")
	}
	req.Header.Set("Content-Type", "application/json")
	if c.CredentialID != "" {
		if a.Secrets == nil {
			return empty, contracts.Fail("unavailable")
		}
		secret, e := a.Secrets.Resolve(bounded, c.CredentialID)
		if e != nil {
			return empty, e
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
		if err := ProviderContextError(bounded); err != nil {
			return empty, err
		}
		return empty, contracts.Condition("unavailable", "provider_unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		reason, code := "provider_failure", "operation_failed"
		switch res.StatusCode {
		case 401, 403:
			reason, code = "provider_auth", "unavailable"
		case 404:
			reason, code = "provider_missing", "unavailable"
		case 429, 503:
			reason, code = "provider_busy", "unavailable"
		case 301, 302, 303, 307, 308:
			reason, code = "provider_redirect", "unavailable"
		}
		return empty, contracts.Condition(code, reason)
	}
	raw, e = io.ReadAll(io.LimitReader(res.Body, int64(c.Limits.MaxResponseBytes)+1))
	if err := ProviderContextError(bounded); err != nil {
		return empty, err
	}
	if len(raw) > c.Limits.MaxResponseBytes {
		return empty, contracts.Condition("output_limit", "provider_response_limit")
	}
	if e != nil {
		return empty, contracts.Condition("operation_failed", "provider_protocol")
	}
	return ParseProvider(raw, c.Adapter)
}
