// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/shruggietech/insonic/internal/backup"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
)

type backupArguments struct {
	action, directory, mode string
}

func parseBackup(args []string) (backupArguments, error) {
	var result backupArguments
	if len(args) < 2 {
		return result, contracts.Fail("invalid_request")
	}
	result.action, result.directory, result.mode = args[0], args[1], "self-contained"
	if result.directory == "" || result.directory[0] == '-' {
		return backupArguments{}, contracts.Fail("invalid_request")
	}
	switch result.action {
	case "create":
		if len(args) == 4 && args[2] == "--mode" && (args[3] == "reference-only" || args[3] == "self-contained") {
			result.mode = args[3]
		} else if len(args) != 2 {
			return backupArguments{}, contracts.Fail("invalid_request")
		}
	case "show", "verify", "restore", "release":
		if len(args) != 2 {
			return backupArguments{}, contracts.Fail("invalid_request")
		}
	default:
		return backupArguments{}, contracts.Fail("invalid_request")
	}
	if result.action == "release" && contracts.ValidID(result.directory) {
		return result, nil
	}
	var err error
	result.directory, err = filepath.Abs(result.directory)
	if err != nil {
		return backupArguments{}, contracts.Fail("invalid_request")
	}
	return result, nil
}

// Maintenance uses the same catalog/artifact contracts without starting a
// processing owner, which could resume imported work before restore commits.
func backupCommand(ctx context.Context, w *workspace.Workspace, args []string, requestID string, machine bool) int {
	parsed, err := parseBackup(args)
	if err != nil {
		return output(nil, err, machine)
	}
	respond := func(result any, err error) int {
		if err != nil {
			return output(nil, err, machine)
		}
		return output(contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, Result: result}, nil, machine)
	}
	if parsed.action == "show" {
		manifest, err := backup.Inspect(ctx, parsed.directory)
		return respond(manifest, err)
	}
	lock, err := w.Lock()
	if err != nil {
		return output(nil, err, machine)
	}
	defer lock.Unlock()
	var input *credentialcmd.Input
	mode, err := secrets.ReadSelection(w)
	if err != nil {
		return output(nil, err, machine)
	}
	if mode == "vault" {
		input, err = credentialcmd.ReadInput(os.Stdin)
		if err != nil {
			return output(nil, err, machine)
		}
		defer input.Close()
	}
	provider, err := credentialcmd.Provider(w, input)
	if err != nil {
		return output(nil, err, machine)
	}
	defer provider.Close()
	if parsed.action == "restore" {
		manifest, err := backup.RestoreWorkspace(ctx, w, provider, parsed.directory)
		return respond(manifest, err)
	}
	if parsed.action == "verify" {
		manifest, err := backup.Verify(ctx, parsed.directory, provider)
		return respond(manifest, err)
	}
	store, err := catalog.OpenWorkspace(ctx, w, provider, false)
	if err != nil {
		return output(nil, err, machine)
	}
	defer store.Close()
	if parsed.action == "release" {
		var err error
		if contracts.ValidID(parsed.directory) {
			err = backup.ReleaseID(ctx, store, parsed.directory)
		} else {
			err = backup.Release(ctx, store, parsed.directory)
		}
		return respond(map[string]any{"released": err == nil}, err)
	}
	manifest, err := backup.Create(ctx, w, store, provider, backup.CreateOptions{ID: requestID, Directory: parsed.directory, Mode: parsed.mode})
	return respond(manifest, err)
}
