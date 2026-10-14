// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/backendsetup"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/firstsetup"
	local "github.com/shruggietech/insonic/internal/runtime"
	"io"
	"os"
	"os/signal"
	"time"
)

func setupBootstrapCommand(args []string, id string, machine bool) int {
	if len(args) < 2 {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	if args[1] == "location" {
		path := ""
		if len(args) == 3 {
			path = args[2]
		} else if len(args) != 2 {
			return output(nil, contracts.Fail("invalid_request"), machine)
		}
		v, e := firstsetup.Preview(path)
		return output(v, e, machine)
	}
	s, e := firstsetup.ProfileStore()
	if e != nil {
		return output(nil, e, machine)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx, bound := context.WithTimeout(ctx, 35*time.Second)
	defer bound()
	if args[1] == "draft" {
		if len(args) == 3 && args[2] == "show" {
			v, e := s.Show()
			return output(v, e, machine)
		}
		if len(args) != 5 || args[2] != "save" || args[3] != "--input" {
			return output(nil, contracts.Fail("invalid_request"), machine)
		}
		file, e := os.Open(args[4])
		if e != nil {
			return output(nil, contracts.Condition("unavailable", "input_missing"), machine)
		}
		raw, e := io.ReadAll(io.LimitReader(file, (128<<10)+1))
		file.Close()
		if e != nil || len(raw) > 128<<10 {
			return output(nil, contracts.Fail("input_limit"), machine)
		}
		v, e := s.Save(ctx, id, raw)
		return output(v, e, machine)
	}
	if args[1] != "provision" {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	mode := ""
	protected := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--secret-input":
			protected = true
		case "--credential-backend":
			if i+1 >= len(args) {
				return output(nil, contracts.Fail("invalid_request"), machine)
			}
			i++
			mode = args[i]
		default:
			return output(nil, contracts.Fail("invalid_request"), machine)
		}
	}
	if mode != "" && mode != "native" && mode != "vault" && mode != "session" {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	draft, e := s.Show()
	if e != nil {
		return output(nil, e, machine)
	}
	if draft.Plan == nil {
		return output(nil, &contracts.Error{Code: "not_found", Message: "Save a setup draft with location and backend choices before provisioning."}, machine)
	}
	if mode == "" {
		mode = draft.CredentialBackend
	} else if mode != draft.CredentialBackend {
		planned, _ := json.Marshal(draft.Plan)
		raw, _ := json.Marshal(firstsetup.BootstrapInput{ExpectedRevision: draft.Revision, Path: draft.Path, Plan: planned, Stage: draft.Stage, Status: draft.Status, CredentialBackend: mode})
		draft, e = s.Save(ctx, id, raw)
		if e != nil {
			return output(nil, e, machine)
		}
	}
	var input *credentialcmd.Input
	var secretRaw []byte
	if protected {
		secretRaw, e = io.ReadAll(io.LimitReader(os.Stdin, credentialcmd.MaxInputBytes+1))
		if e != nil || len(secretRaw) > credentialcmd.MaxInputBytes {
			return output(nil, contracts.Fail("input_limit"), machine)
		}
		defer clear(secretRaw)
		input, e = credentialcmd.ReadInput(bytes.NewReader(secretRaw))
		if e != nil {
			return output(nil, e, machine)
		}
		defer input.Close()
	}
	w, e := backendsetup.ProvisionMode(ctx, draft.Path, *draft.Plan, input, mode)
	if e != nil {
		return output(nil, e, machine)
	}
	operatorContext.Root = w.Root
	responseContext.WorkspaceID = w.Config.WorkspaceID
	if input != nil && (mode == "vault" || mode == "session") {
		if e = local.EnsureInput(ctx, w, "", secretRaw); e != nil {
			return output(map[string]any{"configuration": w.Config, "path": w.Root, "provisioned": true, "runtime_ready": false}, e, machine)
		}
	}
	return output(map[string]any{"configuration": w.Config, "workspace_id": w.Config.WorkspaceID, "path": w.Root, "provisioned": true, "setup_action": "setup show"}, nil, machine)
}
