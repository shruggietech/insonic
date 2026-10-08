// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
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
		fmt.Println("insonic credentials select native/vault/session | credentials add/replace/delete/status <UUID> | credentials unlock/load (protected stdin JSON, saved values never returned)")
		fmt.Println("insonic pipelines/speakers/terms list [--input JSON] | pipelines/speakers/terms show <UUID> | pipelines/speakers/terms set <UUID> --input JSON | pipelines inspect <UUID> [--input JSON] | speakers aliases <UUID> [--input JSON] | speakers select/diagnostics <UUID> [--input JSON] | terms compile [--input JSON]")
		fmt.Println("insonic processing tools <configuration.json> | recordings show/document/mappings <media-id> | recordings process/assemble/map-speaker/export <media-id> --input <JSON>")
		fmt.Println("insonic media tools <configuration.json> | media import <files...> or --manifest <CSV/JSON> [--reference] [--originated-at/on VALUE] [--timezone ZONE] | media list/show/metadata/raw/refresh/set-origin/relocate | models register/acquire <manifest> | models list/show/verify/materialize | work list/show/wait/cancel/retry")
		fmt.Println("insonic settings show | settings set --input JSON | recordings cues <media-id> [--input JSON] | media playback/playback-check/playback-close <media-id> --input JSON")
		fmt.Println("insonic media capture <media-id> --input JSON (exact current metadata/facts/date byte pages)")
		fmt.Println("insonic graph capabilities/status/publish/rebuild | evidence extract/show <media-id> [--input JSON] | query run/explain/validate --input JSON | query run --saved <query-id> [--revision N] | query list/show/save | timeline calendar/recording | views show/save (mutation definitions use --input JSON)")
		fmt.Println("insonic work wait <work-id> [--timeout-ms 30000] (one-process bounded wait; terminal failures preserve state and exit nonzero)")
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
	if (len(positional) == 2 || len(positional) == 3 && positional[2] == "--secret-input") && positional[0] == "runtime" && positional[1] == "serve" {
		var input *credentialcmd.Input
		if len(positional) == 3 {
			input, err = credentialcmd.ReadInput(os.Stdin)
			if err != nil {
				return output(nil, err, machine)
			}
			defer input.Close()
		}
		return output(nil, local.Serve(ctx, w, local.Options{Input: input}), machine)
	}
	if len(positional) >= 2 && positional[0] == "credentials" {
		return credentialCommand(ctx, w, positional[1:], machine)
	}
	if len(positional) >= 2 && positional[0] == "work" && positional[1] == "wait" {
		return workWaitCommand(ctx, w, positional, requestID, machine)
	}
	if len(positional) == 3 && positional[0] == "media" && positional[1] == "tools" {
		return toolsCommand(w, positional[2], machine)
	}
	if len(positional) == 3 && positional[0] == "processing" && positional[1] == "tools" {
		return processingToolsCommand(w, positional[2], machine)
	}
	if len(positional) >= 2 && positional[0] == "catalog" && positional[1] != "show" {
		return catalogCommand(ctx, w, positional[1:], machine)
	}
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: requestID}
	if positional[0] == "search" || (len(positional) > 1 && positional[0] == "query" && positional[1] == "native") {
		req.Operation, req.ItemID, req.Data, err = parseExploreAlias(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if len(positional) > 1 && contracts.ExploreOperation(positional[0]+"."+positional[1]) {
		req.Operation, req.ItemID, req.Data, err = parseExplore(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if len(positional) > 1 && contracts.DesktopOperation(positional[0]+"."+positional[1]) {
		req.Operation, req.ItemID, req.Data, err = parseDesktopOperation(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if positional[0] == "recordings" {
		req.Operation, req.ItemID, req.Data, err = parseRecording(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if positional[0] == "pipelines" || positional[0] == "speakers" || positional[0] == "terms" {
		req.Operation, req.ItemID, req.Data, err = parseConfiguration(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if positional[0] == "media" || positional[0] == "models" || positional[0] == "work" {
		req.Operation, req.ItemID, req.Data, err = parseDomain(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if len(positional) == 1 && positional[0] == "doctor" {
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
	if !contracts.ExploreOperation(req.Operation) && !contracts.ConfigurationOperation(req.Operation) && !strings.HasPrefix(req.Operation, "recordings.") && !strings.HasPrefix(req.Operation, "media.") && !strings.HasPrefix(req.Operation, "models.") && !strings.HasPrefix(req.Operation, "work.") && !strings.Contains("|workspace.show|doctor|catalog.show|jobs.start|jobs.show|jobs.history|jobs.cancel|jobs.retry|artifacts.publish|artifacts.show|artifacts.verify|artifacts.materialize|artifacts.reconcile|artifacts.abort|artifacts.retire|artifacts.lease-renew|artifacts.lease-release|artifacts.cache-prune|artifacts.retain|artifacts.release-reference|", "|"+req.Operation+"|") {
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
	store, err := catalog.OpenWorkspace(ctx, w, provider, args[0] == "migrate")
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

func credentialCommand(ctx context.Context, w *workspace.Workspace, args []string, machine bool) int {
	if len(args) == 0 {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	action := args[0]
	if (action == "unlock" || action == "load") && len(args) != 1 {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	if action != "unlock" && action != "load" && (len(args) != 2 || action != "select" && !contracts.ValidID(args[1])) {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	mode, e := secrets.ReadSelection(w)
	if e != nil {
		return output(nil, e, machine)
	}
	raw := []byte(`{}`)
	if action == "add" || action == "replace" || action == "unlock" || action == "load" {
		raw, e = io.ReadAll(io.LimitReader(os.Stdin, credentialcmd.MaxInputBytes+1))
		if e != nil || len(raw) > credentialcmd.MaxInputBytes {
			return output(nil, contracts.Fail("invalid_request"), machine)
		}
	}
	defer clear(raw)
	input, e := credentialcmd.ReadInput(bytes.NewReader(raw))
	if e != nil {
		return output(nil, e, machine)
	}
	defer input.Close()
	if (action == "unlock" && (mode != "vault" || len(input.Passphrase) == 0)) || (action == "load" && (mode == "native" || mode == "vault" && len(input.Passphrase) == 0 || mode == "session" && input.Session == nil)) {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	input.Close()
	if action == "unlock" || action == "load" {
		e = local.EnsureInput(ctx, w, "", raw)
		return output(map[string]any{"backend": mode, "state": "unlocked"}, e, machine)
	}
	data, _ := json.Marshal(map[string]any{"arguments": args, "input": json.RawMessage(raw)})
	defer clear(data)
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "credentials." + action, Data: data}
	if out, e := local.Call(ctx, w, req); e == nil {
		var err error
		if out.Error != nil {
			err = out.Error
		}
		return output(out, err, machine)
	}
	if mode == "session" && action != "select" && action != "status" {
		if e = local.Ensure(ctx, w, ""); e != nil {
			return output(nil, e, machine)
		}
		out, e := local.Call(ctx, w, req)
		if e == nil && out.Error != nil {
			e = out.Error
		}
		return output(out, e, machine)
	}
	if action == "delete" && mode == "vault" {
		out, e := credentialcmd.Execute(ctx, w, args, os.Stdin)
		return output(out, e, machine)
	}
	out, e := credentialcmd.Execute(ctx, w, args, bytes.NewReader(raw))
	return output(out, e, machine)
}
func toolsCommand(w *workspace.Workspace, path string, machine bool) int {
	return configureToolsCommand(w, path, "media_tools", machine)
}
