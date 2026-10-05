// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

func execute(args []string) int {
	root := ""
	machine := false
	positional := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			machine = true
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
		fmt.Println("insonic workspace init <directory> | workspace show | doctor | jobs start <milliseconds> | jobs show/cancel/retry <job-id> [--workspace <directory>] [--json]")
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
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID()}
	if len(positional) == 1 && positional[0] == "doctor" {
		req.Operation = "doctor"
	} else if len(positional) == 2 && positional[0] == "workspace" && positional[1] == "show" {
		req.Operation = "workspace.show"
	} else if len(positional) == 3 && positional[0] == "jobs" {
		req.Operation = "jobs." + positional[1]
		if positional[1] == "start" {
			req.DurationMS, err = strconv.Atoi(positional[2])
			if err != nil {
				return output(nil, contracts.Fail("invalid_request"), machine)
			}
		} else {
			req.JobID = positional[2]
		}
	} else {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	if !strings.Contains("|workspace.show|doctor|jobs.start|jobs.show|jobs.cancel|jobs.retry|", "|"+req.Operation+"|") {
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
