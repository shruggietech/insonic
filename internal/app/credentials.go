// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
)

// A stable provider reference lets catalog/storage use the currently unlocked
// manager without serializing credentials or reopening adapter contracts.
type liveSecrets struct {
	mu      sync.RWMutex
	manager *secrets.Manager
}

func (l *liveSecrets) Resolve(ctx context.Context, id string) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.manager.Resolve(ctx, id)
}
func (l *liveSecrets) Status(ctx context.Context, id string) (string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.manager.Status(ctx, id)
}
func (l *liveSecrets) Close() { l.mu.Lock(); defer l.mu.Unlock(); l.manager.Close() }
func Bootstrap(w *workspace.Workspace, in *credentialcmd.Input) (contracts.SecretProvider, error) {
	m, e := credentialcmd.Provider(w, in)
	if e != nil {
		return nil, e
	}
	return &liveSecrets{manager: m}, nil
}

func (a *App) credentialDispatch(req contracts.Request) (any, error) {
	if a.secretOwner == nil {
		return nil, contracts.Fail("unavailable")
	}
	var envelope struct {
		Arguments []string        `json:"arguments"`
		Input     json.RawMessage `json:"input,omitempty"`
	}
	if strictPayload(req.Data, &envelope) != nil || len(envelope.Arguments) == 0 || req.Operation != "credentials."+envelope.Arguments[0] {
		return nil, contracts.Fail("invalid_request")
	}
	if envelope.Arguments[0] == "status" {
		envelope.Input = nil
	}
	input, e := credentialcmd.ReadInput(bytes.NewReader(envelope.Input))
	if e != nil {
		return nil, e
	}
	defer input.Close()
	defer clear(req.Data)
	l := a.secretOwner
	l.mu.Lock()
	defer l.mu.Unlock()
	if envelope.Arguments[0] == "select" {
		if len(envelope.Arguments) != 2 {
			return nil, contracts.Fail("invalid_request")
		}
		m, e := credentialcmd.ProviderMode(a.Workspace, envelope.Arguments[1], input)
		if e != nil {
			return nil, e
		}
		if e = secrets.Select(a.Workspace, envelope.Arguments[1]); e != nil {
			m.Close()
			return nil, e
		}
		l.manager.Close()
		l.manager = m
		return map[string]any{"backend": envelope.Arguments[1], "selected": true}, nil
	}
	if envelope.Arguments[0] == "unlock" || envelope.Arguments[0] == "load" {
		mode, e := secrets.ReadSelection(a.Workspace)
		if e != nil || (envelope.Arguments[0] == "unlock" && mode != "vault") || (mode == "vault" && len(input.Passphrase) == 0) || (mode == "session" && input.Session == nil) || mode == "native" {
			return nil, contracts.Fail("invalid_request")
		}
		m, e := credentialcmd.Provider(a.Workspace, input)
		if e != nil {
			return nil, e
		}
		l.manager.Close()
		l.manager = m
		return map[string]any{"backend": mode, "state": "unlocked"}, nil
	}
	if len(input.Passphrase) > 0 {
		m, e := credentialcmd.Provider(a.Workspace, input)
		if e != nil {
			return nil, e
		}
		l.manager.Close()
		l.manager = m
	}
	return credentialcmd.ExecuteWithManager(a.ctx, l.manager, envelope.Arguments, input)
}
