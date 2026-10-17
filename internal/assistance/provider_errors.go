// SPDX-License-Identifier: Apache-2.0
package assistance

import (
	"context"
	"errors"
	"github.com/shruggietech/insonic/internal/contracts"
)

// ProviderContextError separates an expired provider budget from caller cancellation.
func ProviderContextError(ctx context.Context) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return contracts.Condition("unavailable", "provider_timeout")
	}
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	return nil
}

// ParseProvider reports untrusted provider output without echoing parser details.
func ParseProvider(raw []byte, adapter string) (Proposal, error) {
	p, err := Parse(raw)
	if err != nil {
		reason := "provider_protocol"
		if adapter == "ollama" {
			reason = "ollama_protocol"
		}
		return Proposal{}, contracts.Condition("operation_failed", reason)
	}
	return p, nil
}
