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
	var empty Proposal
	if a.Client == nil && InCI() {
		return empty, contracts.Fail("engine_ci_forbidden")
	}
	if ctx.Err() != nil {
		return empty, contracts.Fail("cancelled")
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
	client := http.Client{Timeout: time.Duration(c.Limits.TimeoutMS) * time.Millisecond, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if a.Client != nil {
		client.Transport = a.Client.Transport
	}
	defer client.CloseIdleConnections()
	res, e := client.Do(req)
	if e != nil {
		if bounded.Err() != nil {
			return empty, contracts.Fail("cancelled")
		}
		return empty, contracts.Fail("unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return empty, contracts.Fail("operation_failed")
	}
	raw, e = io.ReadAll(io.LimitReader(res.Body, int64(c.Limits.MaxResponseBytes)+1))
	if bounded.Err() != nil {
		return empty, contracts.Fail("cancelled")
	}
	if e != nil || len(raw) > c.Limits.MaxResponseBytes {
		return empty, contracts.Fail("output_limit")
	}
	return Parse(raw)
}
