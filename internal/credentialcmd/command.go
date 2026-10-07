// SPDX-License-Identifier: Apache-2.0
// Package credentialcmd shares credential bootstrap between CLI and desktop.
// It does not open a catalog or serialize transient credential inputs.
package credentialcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
)

const MaxInputBytes = 1 << 20

// Input is a protected-channel message, not a persisted runtime request.
// Values and passphrases are intentionally excluded from JSON serialization.
type Input struct {
	Value      []byte            `json:"-"`
	Passphrase []byte            `json:"-"`
	Session    map[string][]byte `json:"-"`
}

func (in *Input) Close() {
	if in == nil {
		return
	}
	clear(in.Value)
	clear(in.Passphrase)
	for id, value := range in.Session {
		clear(value)
		delete(in.Session, id)
	}
	in.Value = nil
	in.Passphrase = nil
	in.Session = nil
}

func ReadInput(reader io.Reader) (*Input, error) {
	if reader == nil {
		return &Input{}, nil
	}
	raw, e := io.ReadAll(io.LimitReader(reader, MaxInputBytes+1))
	defer clear(raw)
	if e != nil || len(raw) > MaxInputBytes {
		return nil, contracts.Fail("invalid_request")
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return &Input{}, nil
	}
	if validate(raw) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return nil, contracts.Fail("invalid_request")
	}
	defer func() {
		for _, value := range fields {
			clear(value)
		}
	}()
	in := &Input{}
	valid := false
	defer func() {
		if !valid {
			in.Close()
		}
	}()
	for field, value := range fields {
		switch field {
		case "value":
			in.Value, e = credentialValue(value)
		case "passphrase":
			var pass string
			e = json.Unmarshal(value, &pass)
			if e == nil {
				in.Passphrase = []byte(pass)
				if len(in.Passphrase) == 0 || len(in.Passphrase) > secrets.MaxValueBytes {
					e = contracts.Fail("invalid_request")
				}
			}
		case "session":
			in.Session, e = sessionValues(value)
		default:
			e = contracts.Fail("invalid_request")
		}
		if e != nil {
			return nil, contracts.Fail("invalid_request")
		}
	}
	valid = true
	return in, nil
}
func credentialValue(raw []byte) ([]byte, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, contracts.Fail("invalid_request")
	}
	var value []byte
	switch raw[0] {
	case '"':
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		value = []byte(text)
	case '{':
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil || fields == nil {
			return nil, contracts.Fail("invalid_request")
		}
		value = append([]byte(nil), raw...)
	default:
		return nil, contracts.Fail("invalid_request")
	}
	if len(value) == 0 || len(value) > secrets.MaxValueBytes {
		clear(value)
		return nil, contracts.Fail("invalid_request")
	}
	return value, nil
}
func sessionValues(raw []byte) (map[string][]byte, error) {
	if validate(raw) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil || len(fields) > 256 {
		return nil, contracts.Fail("invalid_request")
	}
	defer func() {
		for _, value := range fields {
			clear(value)
		}
	}()
	values := map[string][]byte{}
	for id, field := range fields {
		if !contracts.ValidID(id) {
			for _, v := range values {
				clear(v)
			}
			return nil, contracts.Fail("invalid_request")
		}
		value, e := credentialValue(field)
		if e != nil {
			for _, v := range values {
				clear(v)
			}
			return nil, e
		}
		values[id] = value
	}
	return values, nil
}

// Provider reads only the selected backend. A session environment mapping is
// considered solely when session mode was explicitly selected for this workspace.
// The caller owns the returned manager and must keep it alive for adapter use.
func Provider(w *workspace.Workspace, in *Input) (*secrets.Manager, error) {
	mode, e := secrets.ReadSelection(w)
	if e != nil {
		return nil, e
	}
	return ProviderMode(w, mode, in)
}

// ProviderMode validates a proposed backend before its selection is persisted.
// This preserves the current selection when explicit input cannot be admitted.
func ProviderMode(w *workspace.Workspace, mode string, in *Input) (*secrets.Manager, error) {
	if mode != "native" && mode != "vault" && mode != "session" {
		return nil, contracts.Fail("invalid_request")
	}
	opts := secrets.Options{Mode: mode}
	if in != nil {
		opts.Passphrase = in.Passphrase
		opts.Session = in.Session
	}
	var transient map[string][]byte
	var e error
	if mode == "session" && opts.Session == nil {
		raw := os.Getenv("INSONIC_SESSION_CREDENTIALS")
		if len(raw) > MaxInputBytes {
			return nil, contracts.Fail("invalid_request")
		}
		if raw != "" {
			transient, e = sessionValues([]byte(raw))
			if e != nil {
				return nil, e
			}
			opts.Session = transient
			defer func() {
				for _, v := range transient {
					clear(v)
				}
			}()
		}
	}
	return secrets.Open(w, opts)
}

func Execute(ctx context.Context, w *workspace.Workspace, args []string, stdin io.Reader) (any, error) {
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	if len(args) == 2 && args[0] == "select" {
		manager, e := ProviderMode(w, args[1], nil)
		if e != nil {
			return nil, e
		}
		defer manager.Close()
		if e := secrets.Select(w, args[1]); e != nil {
			return nil, e
		}
		return map[string]any{"schema_version": contracts.Version, "backend": args[1], "selected": true}, nil
	}
	if !validCommand(args) {
		return nil, contracts.Fail("invalid_request")
	}
	mode, e := secrets.ReadSelection(w)
	if e != nil {
		return nil, e
	}
	// A short-lived CLI manager cannot persist session changes. Session mutations
	// must use the runtime's manager through ExecuteWithManager instead.
	if mode == "session" && (args[0] == "add" || args[0] == "replace" || args[0] == "delete") {
		return nil, contracts.Fail("unavailable")
	}
	input := &Input{}
	if args[0] != "status" && (args[0] == "add" || args[0] == "replace" || args[0] == "unlock" || mode == "vault") {
		input, e = ReadInput(stdin)
		if e != nil {
			return nil, e
		}
	}
	defer input.Close()
	manager, e := Provider(w, input)
	if e != nil {
		return nil, e
	}
	defer manager.Close()
	return ExecuteWithManager(ctx, manager, args, input)
}
func validCommand(args []string) bool {
	if len(args) == 1 && args[0] == "unlock" {
		return true
	}
	if len(args) != 2 || !contracts.ValidID(args[1]) {
		return false
	}
	return args[0] == "add" || args[0] == "replace" || args[0] == "delete" || args[0] == "status"
}

// ExecuteWithManager keeps explicit session credentials in the live runtime.
// It can also be used for persistent stores without changing command behavior.
func ExecuteWithManager(ctx context.Context, manager *secrets.Manager, args []string, input *Input) (any, error) {
	if manager == nil || !validCommand(args) {
		return nil, contracts.Fail("invalid_request")
	}
	if input == nil {
		input = &Input{}
	}
	if args[0] == "unlock" {
		status, e := manager.Inspect(ctx, contracts.ID())
		if e != nil {
			return nil, e
		}
		if status.Backend != "vault" || len(input.Passphrase) == 0 || status.State == "rejected" {
			return nil, contracts.Fail("unavailable")
		}
		return map[string]any{"schema_version": contracts.Version, "backend": "vault", "state": "validated"}, nil
	}
	id := args[1]
	var e error
	switch args[0] {
	case "add":
		e = manager.Add(ctx, id, input.Value)
	case "replace":
		e = manager.Replace(ctx, id, input.Value)
	case "delete":
		e = manager.Delete(ctx, id)
	}
	if e != nil {
		return nil, e
	}
	return manager.Inspect(ctx, id)
}

// validate rejects duplicate JSON keys before unmarshalling a protected input.
func validate(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e := unique(d, 0); e != nil {
		return e
	}
	if _, e := d.Token(); e != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func unique(d *json.Decoder, depth int) error {
	if depth > 64 {
		return contracts.Fail("invalid_request")
	}
	token, e := d.Token()
	if e != nil {
		return e
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	if delimiter != '{' && delimiter != '[' {
		return contracts.Fail("invalid_request")
	}
	seen := map[string]bool{}
	for d.More() {
		if delimiter == '{' {
			token, e := d.Token()
			if e != nil {
				return e
			}
			name, ok := token.(string)
			if !ok || seen[name] {
				return contracts.Fail("invalid_request")
			}
			seen[name] = true
		}
		if e := unique(d, depth+1); e != nil {
			return e
		}
	}
	token, e = d.Token()
	if e != nil {
		return e
	}
	if delimiter == '{' && token != json.Delim('}') || delimiter == '[' && token != json.Delim(']') {
		return contracts.Fail("invalid_request")
	}
	return nil
}
