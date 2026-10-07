// SPDX-License-Identifier: Apache-2.0
// This qualification command writes only randomized fixture credentials in its
// own temporary workspace and prints fixed status fields, never saved values.
package main

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/process"
	"github.com/shruggietech/insonic/internal/secrets"
	"github.com/shruggietech/insonic/internal/workspace"
)

func qualify() (outErr error) {
	directory, e := os.MkdirTemp("", "insonic-secret-lifecycle-")
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer os.RemoveAll(directory)
	restore, e := nativeFixture(directory)
	if e != nil {
		return e
	}
	defer func() {
		if e := restore(); outErr == nil {
			outErr = e
		}
	}()
	w, e := workspace.Init(directory, "Native credentials qualification")
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	id := contracts.ID()
	manager, e := secrets.Open(w, secrets.Options{Mode: "native"})
	if e != nil {
		return e
	}
	defer func() {
		if manager != nil {
			manager.Close()
		}
		cleanup, openErr := secrets.Open(w, secrets.Options{Mode: "native"})
		if openErr == nil {
			cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			cleanup.Delete(cleanupCtx, id)
			cleanup.Close()
		}
	}()
	if e = manager.Add(ctx, id, []byte("insonic-random-workspace-native-fixture")); e != nil {
		return e
	}
	if state, e := manager.Status(ctx, id); e != nil || state != "configured" {
		return contracts.Fail("operation_failed")
	}
	if e = manager.Close(); e != nil {
		return e
	}
	if e = restartProbe(ctx, directory, id, false); e != nil {
		return e
	}
	manager, e = secrets.Open(w, secrets.Options{Mode: "native"})
	if e != nil {
		return e
	}
	value, e := manager.Resolve(ctx, id)
	if e != nil {
		return e
	}
	valid := string(value) == "insonic-random-workspace-native-fixture"
	clear(value)
	if !valid {
		return contracts.Fail("operation_failed")
	}
	if e = manager.Replace(ctx, id, []byte("insonic-native-replacement-fixture")); e != nil {
		return e
	}
	if e = restartProbe(ctx, directory, id, true); e != nil {
		return e
	}
	value, e = manager.Resolve(ctx, id)
	if e != nil {
		return e
	}
	valid = string(value) == "insonic-native-replacement-fixture"
	clear(value)
	if !valid {
		return contracts.Fail("operation_failed")
	}
	if e = manager.Delete(ctx, id); e != nil {
		return e
	}
	if state, e := manager.Status(ctx, id); e != nil || state != "missing" {
		return contracts.Fail("operation_failed")
	}
	if value, e = manager.Resolve(ctx, id); e == nil {
		clear(value)
		return contracts.Fail("operation_failed")
	}
	return nil
}
func restartProbe(ctx context.Context, directory, id string, replaced bool) error {
	executable, e := os.Executable()
	if e != nil {
		return contracts.Fail("unavailable")
	}
	phase := "--read-back"
	if replaced {
		phase = "--read-back-replaced"
	}
	output, e := process.Run(ctx, process.Spec{Executable: executable, Args: []string{phase, directory, id}, MaxOutput: 2048})
	if e != nil {
		return e
	}
	var result map[string]string
	if json.Unmarshal(output.Output, &result) != nil || result["native_credential_lifecycle"] != "passed" {
		return contracts.Fail("operation_failed")
	}
	return nil
}
func readBack(directory, id string, replaced bool) error {
	if !contracts.ValidID(id) {
		return contracts.Fail("invalid_request")
	}
	w, e := workspace.Open(directory)
	if e != nil {
		return e
	}
	manager, e := secrets.Open(w, secrets.Options{Mode: "native"})
	if e != nil {
		return e
	}
	defer manager.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	value, e := manager.Resolve(ctx, id)
	if e != nil {
		return e
	}
	defer clear(value)
	expected := "insonic-random-workspace-native-fixture"
	if replaced {
		expected = "insonic-native-replacement-fixture"
	}
	if string(value) != expected {
		return contracts.Fail("operation_failed")
	}
	return nil
}
func main() {
	result := map[string]any{"schema_version": contracts.Version, "platform": runtime.GOOS, "native_credential_lifecycle": "passed", "credential_values": "not-returned", "cross_process_restart": "passed"}
	var e error
	if len(os.Args) == 4 && (os.Args[1] == "--read-back" || os.Args[1] == "--read-back-replaced") {
		e = readBack(os.Args[2], os.Args[3], os.Args[1] == "--read-back-replaced")
	} else if len(os.Args) == 1 {
		e = qualify()
	} else {
		e = contracts.Fail("invalid_request")
	}
	if e != nil {
		result["native_credential_lifecycle"] = "failed"
		result["error_code"] = "unavailable"
		json.NewEncoder(os.Stdout).Encode(result)
		os.Exit(1)
	}
	json.NewEncoder(os.Stdout).Encode(result)
}
