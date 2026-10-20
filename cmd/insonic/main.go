// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"github.com/shruggietech/insonic/internal/diagnostics"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func execute(args []string) int {
	parsed, parseErr := parseInvocation(args)
	root, machine, requestID := parsed.Root, parsed.Machine, parsed.RequestID
	operatorContext = parsed
	startupReported = false
	responseContext = contracts.Response{Kind: "runtime-response", Version: contracts.Version}
	if contracts.ValidID(requestID) {
		responseContext.RequestID = requestID
	}
	if parseErr != nil {
		operatorContext.StartupID = ""
		return output(nil, parseErr, machine)
	}
	if err := workspace.RequireSupportedHost(); err != nil {
		return output(nil, err, machine)
	}
	if parsed.Help {
		if machine {
			return output(map[string]any{"command": strings.Join(parsed.Path, " "), "help": commandHelp(parsed.Path)}, nil, true)
		}
		return writeHuman(commandHelp(parsed.Path), os.Stdout)
	}
	positional := parsed.Args
	if len(positional) == 1 && positional[0] == "version" {
		if machine {
			return output(map[string]string{"version": contracts.Version}, nil, true)
		}
		return writeHuman(contracts.Version+"\n", os.Stdout)
	}
	operatorLog, errLog := diagnostics.Open(parsed.LogFile, parsed.LogLevel)
	if errLog != nil {
		return output(nil, errLog, machine)
	}
	activeLog = operatorLog
	defer func() {
		if activeLog != nil {
			activeLog.Close()
			activeLog = nil
		}
	}()
	commandStarted = time.Now()
	if e := activeLog.Write("debug", diagnosticEvent("")); e != nil {
		return output(nil, e, machine)
	}
	if len(positional) >= 2 && positional[0] == "setup" && (positional[1] == "location" || positional[1] == "draft" || positional[1] == "provision") {
		return setupBootstrapCommand(positional, requestID, machine)
	}
	if len(positional) >= 2 && positional[0] == "workspace" && (positional[1] == "plan" || positional[1] == "preflight" || positional[1] == "provision") {
		return backendCommand(positional, machine)
	}
	if len(positional) == 3 && positional[0] == "workspace" && positional[1] == "init" {
		w, err := workspace.Init(positional[2], "Workspace")
		if err != nil {
			return output(nil, err, machine)
		}
		operatorContext.Root = w.Root
		responseContext.WorkspaceID = w.Config.WorkspaceID
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
		var typed *contracts.Error
		if errors.As(err, &typed) && typed.Reason == "workspace_missing" {
			err = contracts.Condition("unavailable", "workspace_missing")
		}
		return output(nil, err, machine)
	}
	operatorContext.Root = w.Root
	responseContext.WorkspaceID = w.Config.WorkspaceID
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
		return output(nil, local.Serve(ctx, w, local.Options{Input: input, ReportReady: func() {
			if parsed.StartupID != "" {
				emitStartup(nil)
			}
		}}), machine)
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
	if len(positional) >= 1 && positional[0] == "backup" {
		return backupCommand(ctx, w, positional[1:], requestID, machine)
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
	} else if len(positional) > 1 && (contracts.DesktopOperation(positional[0]+"."+positional[1]) || positional[0] == "settings" && positional[1] == "setup") {
		req.Operation, req.ItemID, req.Data, err = parseDesktopOperation(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if positional[0] == "recordings" {
		req.Operation, req.ItemID, req.Data, err = parseRecording(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if positional[0] == "import-presets" || positional[0] == "pipelines" || positional[0] == "speakers" || positional[0] == "terms" {
		req.Operation, req.ItemID, req.Data, err = parseConfiguration(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if positional[0] == "workers" || len(positional) > 1 && (positional[0] == "processing" && positional[1] == "readiness" || positional[0] == "models" && (positional[1] == "defaults" || positional[1] == "prepare")) {
		req.Operation, req.ItemID, req.Data, err = parseWorkers(positional)
		if err != nil {
			return output(nil, err, machine)
		}
	} else if positional[0] == "media" || positional[0] == "models" || positional[0] == "work" || positional[0] == "transcript" {
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
	encoded, encodeErr := json.Marshal(req)
	if encodeErr != nil {
		return output(nil, contracts.Condition("invalid_request", "input_invalid"), machine)
	}
	if err := schemas.ValidateRequest(encoded); err != nil {
		return output(nil, err, machine)
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
			return output(nil, contracts.Condition("unavailable", "input_missing"), machine)
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
	if err := exportCatalogFile(args[1], snap); err != nil {
		return output(nil, err, machine)
	}
	return output(map[string]any{"exported": true, "workspace_id": snap.WorkspaceID, "revision": snap.Revision, "digest": snap.Digest}, nil, machine)
}

var responseContext contracts.Response
var activeLog *diagnostics.Logger
var commandStarted time.Time

func diagnosticEvent(state string) diagnostics.Event {
	return diagnostics.Event{Operation: strings.Join(operatorContext.Path, "."), RequestID: responseContext.RequestID, WorkspaceID: responseContext.WorkspaceID, State: state, DurationMS: time.Since(commandStarted).Milliseconds()}
}

func exitCode(err *contracts.Error) int {
	if err == nil {
		return 0
	}
	switch err.Code {
	case "invalid_request":
		return 2
	case "incompatible_version":
		return 3
	case "unavailable":
		return 4
	}
	return 1
}
func writeHuman(text string, writer io.Writer) int {
	if _, err := io.WriteString(writer, text); err != nil {
		fmt.Fprintln(os.Stderr, "unavailable: Could not write command output.")
		return 4
	}
	return 0
}
func output(result any, err error, machine bool) int {
	if operatorContext.StartupID != "" {
		if err != nil && !startupReported {
			emitStartup(err)
		}
		if err != nil {
			return exitCode(contracts.SafeError(err))
		}
		return 0
	}
	if activeLog != nil {
		event := diagnosticEvent("")
		level := "info"
		if err != nil {
			safe := contracts.SafeError(err)
			event.Code, event.Reason = safe.Code, safe.Reason
			level = "error"
		}
		if logErr := activeLog.Write(level, event); logErr != nil {
			err = logErr
		}
		if logErr := activeLog.Close(); logErr != nil && err == nil {
			err = logErr
		}
		activeLog = nil
	}
	return outputTo(result, err, machine, responseContext, os.Stdout, os.Stderr)
}
func outputTo(result any, err error, machine bool, context contracts.Response, stdout, stderr io.Writer) int {
	var typed *contracts.Error
	if err != nil {
		if !errors.As(err, &typed) {
			typed = contracts.Fail("operation_failed")
		}
	}
	response, ok := result.(contracts.Response)
	if !ok {
		response = context
		response.Result = result
	}
	response.Kind = "runtime-response"
	response.Version = contracts.Version
	if response.RequestID == "" {
		response.RequestID = context.RequestID
	}
	if response.WorkspaceID == "" {
		response.WorkspaceID = context.WorkspaceID
	}
	if typed != nil {
		response.Error = typed
	}
	if response.Error != nil {
		typed = response.Error
	}
	if typed != nil {
		response.Result = contracts.ObservedResult(response.Result, typed)
	}
	writer := stdout
	if typed != nil {
		writer = stderr
	}
	if result == nil && typed == nil {
		return 0
	}
	var writeErr error
	if machine {
		writeErr = json.NewEncoder(writer).Encode(response)
	} else {
		var text string
		text, writeErr = renderHuman(response)
		if writeErr == nil {
			_, writeErr = io.WriteString(writer, text)
		}
	}
	if writeErr != nil {
		fmt.Fprintln(stderr, "unavailable: Could not encode or write command output.")
		return 4
	}
	return exitCode(typed)
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
	if action == "add" || action == "replace" || action == "unlock" || action == "load" || action == "delete" && mode == "vault" {
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
	if (action == "delete" && mode == "vault" && len(input.Passphrase) == 0) || (action == "unlock" && (mode != "vault" || len(input.Passphrase) == 0)) || (action == "load" && (mode == "native" || mode == "vault" && len(input.Passphrase) == 0 || mode == "session" && input.Session == nil)) {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	input.Close()
	if action == "unlock" || action == "load" {
		e = local.EnsureInput(ctx, w, "", raw)
		if e != nil {
			return output(nil, e, machine)
		}
		state := "unlocked"
		if mode == "session" {
			state = "loaded"
		}
		return output(map[string]any{"backend": mode, "state": state, "lifetime": "runtime-owner"}, nil, machine)
	}
	data, _ := json.Marshal(map[string]any{"arguments": args, "input": json.RawMessage(raw)})
	defer clear(data)
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: responseContext.RequestID, Operation: "credentials." + action, Data: data}
	if !contracts.ValidID(req.RequestID) {
		req.RequestID = contracts.ID()
	}
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
	out, e := credentialcmd.Execute(ctx, w, args, bytes.NewReader(raw))
	return output(out, e, machine)
}
func toolsCommand(w *workspace.Workspace, path string, machine bool) int {
	return configureToolsCommand(w, path, "media_tools", machine)
}
