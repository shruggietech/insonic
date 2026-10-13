// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"github.com/shruggietech/insonic/internal/backendsetup"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	local "github.com/shruggietech/insonic/internal/runtime"
	"io"
	"os"
	"os/signal"
	"time"
)

func backendCommand(args []string, machine bool) int {
	if len(args) < 2 {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	action := args[1]
	index := 2
	root := ""
	if action != "plan" {
		if len(args) < 3 {
			return output(nil, contracts.Fail("invalid_request"), machine)
		}
		root = args[2]
		index = 3
	}
	file := ""
	protected := false
	mode := "native"
	for index < len(args) {
		switch args[index] {
		case "--input":
			if index+1 >= len(args) {
				return output(nil, contracts.Fail("invalid_request"), machine)
			}
			file = args[index+1]
			index += 2
		case "--secret-input":
			protected = true
			index++
		case "--credential-backend":
			if index+1 >= len(args) {
				return output(nil, contracts.Fail("invalid_request"), machine)
			}
			mode = args[index+1]
			index += 2
		default:
			return output(nil, contracts.Fail("invalid_request"), machine)
		}
	}
	f, e := os.Open(file)
	if e != nil {
		return output(nil, contracts.Condition("unavailable", "input_missing"), machine)
	}
	raw, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	f.Close()
	if e != nil || len(raw) > 1<<20 {
		return output(nil, contracts.Fail("input_limit"), machine)
	}
	c, e := backendsetup.Build(raw)
	if e != nil {
		return output(backendsetup.Failure(e), e, machine)
	}
	if action == "plan" {
		return output(map[string]any{"configuration": c, "provisioned": false}, nil, machine)
	}
	var input *credentialcmd.Input
	secretRaw := []byte(nil)
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
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx, bound := context.WithTimeout(ctx, 35*time.Second)
	defer bound()
	if action == "preflight" {
		out, e := backendsetup.Preflight(ctx, root, c, backendsetup.Transient{Input: input})
		return output(out, e, machine)
	}
	w, e := backendsetup.ProvisionMode(ctx, root, c, input, mode)
	if e != nil {
		return output(nil, e, machine)
	}
	operatorContext.Root = w.Root
	responseContext.WorkspaceID = w.Config.WorkspaceID
	if input != nil && (mode == "vault" || mode == "session") {
		if e = local.EnsureInput(ctx, w, "", secretRaw); e != nil {
			return output(map[string]any{"configuration": w.Config, "provisioned": true, "runtime_ready": false}, e, machine)
		}
	}
	return output(map[string]any{"configuration": w.Config, "provisioned": true}, nil, machine)
}
