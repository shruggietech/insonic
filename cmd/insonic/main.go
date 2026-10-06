// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
)

func execute(args []string) int {
	root := ""
	machine := false
	requestID := contracts.ID()
	positional := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			machine = true
		case "--request-id":
			i++
			if i >= len(args) || !contracts.ValidID(args[i]) {
				return output(nil, contracts.Fail("invalid_request"), machine)
			}
			requestID = args[i]
		case "--workspace":
			i++
			if i >= len(args) {
				return output(nil, contracts.Fail("invalid_request"), machine)
			}
			root = args[i]
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) == 1 && (positional[0] == "version" || positional[0] == "--version") {
		fmt.Println(contracts.Version)
		return 0
	}
	if len(positional) == 0 || (len(positional) == 1 && positional[0] == "--help") {
		fmt.Println("insonic workspace init <directory> | workspace show | doctor | jobs start <milliseconds> | jobs show/cancel/retry <job-id> | jobs history <job-id> [after-generation] | catalog show/migrate | catalog export/restore <file> [--workspace <directory>] [--request-id <UUID>] [--json]")
		fmt.Println("insonic artifacts publish <file> <kind> | artifacts show/verify/reconcile/abort/retire/cache-prune <publication-id> | artifacts materialize <publication-id> [max-bytes] | artifacts lease-renew/lease-release <publication-id> <lease-id> | artifacts retain/release-reference <publication-id> <reference-id>")
		return 0
	}
	if len(positional) == 3 && positional[0] == "workspace" && positional[1] == "init" {
		w, err := workspace.Init(positional[2], "Workspace")
		if err != nil {
			return output(nil, err, machine)
		}
		return output(contracts.Response{Kind: "runtime-response", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, Result: w.Config}, nil, machine)
	}
	var w *workspace.Workspace
	var err error
	if root != "" {
		w, err = workspace.Open(root)
	} else {
		cwd, _ := os.Getwd()
		w, err = workspace.Discover(cwd)
	}
	if err != nil {
		return output(nil, err, machine)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if len(positional) == 2 && positional[0] == "runtime" && positional[1] == "serve" {
		return output(nil, local.Serve(ctx, w, local.Options{}), machine)
	}
	if len(positional) >= 2 && positional[0] == "catalog" && positional[1] != "show" {
		return catalogCommand(ctx, w, positional[1:], machine)
	}
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: requestID}
	if len(positional) == 1 && positional[0] == "doctor" {
		req.Operation = "doctor"
	} else if len(positional) == 2 && positional[0] == "workspace" && positional[1] == "show" {
		req.Operation = "workspace.show"
	} else if len(positional) == 2 && positional[0] == "catalog" && positional[1] == "show" {
		req.Operation = "catalog.show"
	} else if len(positional) >= 3 && positional[0] == "artifacts" {
		req.Operation = "artifacts." + positional[1]
		if positional[1] == "publish" && len(positional) == 4 {
			req.SourcePath, err = filepath.Abs(positional[2])
			if err != nil {
				return output(nil, contracts.Fail("invalid_request"), machine)
			}
			req.ArtifactKind = positional[3]
		} else if positional[1] == "materialize" && len(positional) == 4 {
			req.PublicationID = positional[2]
			req.MaxBytes, err = strconv.ParseInt(positional[3], 10, 64)
			if err != nil || req.MaxBytes < 1 {
				return output(nil, contracts.Fail("invalid_request"), machine)
			}
		} else if (positional[1] == "lease-renew" || positional[1] == "lease-release") && len(positional) == 4 {
			req.PublicationID = positional[2]
			req.LeaseID = positional[3]
		} else if (positional[1] == "retain" || positional[1] == "release-reference") && len(positional) == 4 {
			req.PublicationID = positional[2]
			req.ReferenceID = positional[3]
		} else if len(positional) == 3 {
			req.PublicationID = positional[2]
		} else {
			return output(nil, contracts.Fail("invalid_request"), machine)
		}
	} else if (len(positional) == 3 || (len(positional) == 4 && positional[1] == "history")) && positional[0] == "jobs" {
		req.Operation = "jobs." + positional[1]
		if positional[1] == "start" {
			req.DurationMS, err = strconv.Atoi(positional[2])
			if err != nil {
				return output(nil, contracts.Fail("invalid_request"), machine)
			}
		} else {
			req.JobID = positional[2]
			if len(positional) == 4 {
				req.AfterGeneration, err = strconv.ParseInt(positional[3], 10, 64)
				if err != nil || req.AfterGeneration < 0 {
					return output(nil, contracts.Fail("invalid_request"), machine)
				}
			}
		}
	} else {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	if !strings.Contains("|workspace.show|doctor|catalog.show|jobs.start|jobs.show|jobs.history|jobs.cancel|jobs.retry|artifacts.publish|artifacts.show|artifacts.verify|artifacts.materialize|artifacts.reconcile|artifacts.abort|artifacts.retire|artifacts.lease-renew|artifacts.lease-release|artifacts.cache-prune|artifacts.retain|artifacts.release-reference|", "|"+req.Operation+"|") {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	if err := local.Ensure(ctx, w, ""); err != nil {
		return output(nil, err, machine)
	}
	response, err := local.Call(ctx, w, req)
	if err == nil && response.Error != nil {
		err = response.Error
	}
	return output(response, err, machine)
}

func catalogCommand(ctx context.Context, w *workspace.Workspace, args []string, machine bool) int {
	if (len(args) != 1 || args[0] != "migrate") && (len(args) != 2 || (args[0] != "export" && args[0] != "restore")) {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	if args[0] != "export" {
		lock, err := w.Lock()
		if err != nil {
			return output(nil, err, machine)
		}
		defer lock.Unlock()
	}
	secrets, err := catalog.SessionSecretsFromEnvironment()
	if err != nil {
		return output(nil, err, machine)
	}
	store, err := catalog.OpenWorkspace(ctx, w, secrets, args[0] == "migrate")
	if err != nil {
		return output(nil, err, machine)
	}
	defer store.Close()
	if args[0] == "migrate" {
		status, err := store.Status(ctx)
		return output(status, err, machine)
	}
	if args[0] == "restore" {
		file, err := os.Open(args[1])
		if err != nil {
			return output(nil, contracts.Fail("unavailable"), machine)
		}
		defer file.Close()
		snap, err := catalog.ReadSnapshotReader(file)
		if err == nil {
			err = store.Restore(ctx, snap)
		}
		return output(map[string]any{"restored": err == nil}, err, machine)
	}
	snap, err := store.Export(ctx)
	if err != nil {
		return output(nil, err, machine)
	}
	file, err := os.OpenFile(args[1], os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return output(nil, contracts.Fail("conflict"), machine)
	}
	err = json.NewEncoder(file).Encode(snap)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(args[1])
		return output(nil, contracts.Fail("unavailable"), machine)
	}
	return output(map[string]any{"exported": true, "workspace_id": snap.WorkspaceID, "revision": snap.Revision, "digest": snap.Digest}, nil, machine)
}

func output(result any, err error, machine bool) int {
	if err != nil {
		typed, ok := err.(*contracts.Error)
		if !ok {
			typed = contracts.Fail("operation_failed")
		}
		if machine {
			json.NewEncoder(os.Stderr).Encode(contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: typed})
		} else {
			fmt.Fprintln(os.Stderr, typed.Error())
		}
		if typed.Code == "invalid_request" {
			return 2
		}
		if typed.Code == "incompatible_version" {
			return 3
		}
		if typed.Code == "unavailable" {
			return 4
		}
		return 1
	}
	if result != nil {
		if machine {
			json.NewEncoder(os.Stdout).Encode(result)
		} else {
			data, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(data))
		}
	}
	return 0
}
func main() { os.Exit(execute(os.Args[1:])) }
