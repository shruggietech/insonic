// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
)

// SessionSecrets is explicit transient input, not ambient PostgreSQL routing.
// Native/encrypted persistent credential storage is a separate implementation.
type SessionSecrets map[string]json.RawMessage

func SessionSecretsFromEnvironment() (SessionSecrets, error) {
	raw := os.Getenv("INSONIC_SESSION_CREDENTIALS")
	if raw == "" {
		return nil, nil
	}
	var s SessionSecrets
	if strict([]byte(raw), &s) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	return s, nil
}
func (s SessionSecrets) Resolve(_ context.Context, id string) ([]byte, error) {
	raw, ok := s[id]
	if !ok {
		return nil, contracts.Fail("unavailable")
	}
	return append([]byte{}, raw...), nil
}
func (s SessionSecrets) Status(_ context.Context, id string) (string, error) {
	if _, ok := s[id]; ok {
		return "session", nil
	}
	return "unavailable", nil
}
