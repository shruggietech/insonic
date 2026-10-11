// SPDX-License-Identifier: Apache-2.0
package contracts

import "errors"

var operatorReasons = map[string]string{
	"workspace_missing":         "No workspace was found. Initialize a library or select its directory with --workspace.",
	"input_missing":             "The selected input file is unavailable. Check its path and permissions.",
	"input_invalid":             "The selected input does not match this command's contract. Use contextual --help for the required fields.",
	"revision_stale":            "The selected revision is no longer current. Inspect the item and retry with its current revision.",
	"credential_duplicate":      "This credential ID is already stored. Inspect its status or use credentials replace with the same ID.",
	"credential_capacity":       "The credential store is full. Remove an unused credential before adding another.",
	"credential_locked":         "The saved credential is locked in this manager. Unlock the vault through protected input.",
	"credential_bootstrap":      "Credential bootstrap failed. Check the selected backend, vault file and protected passphrase input; no value was logged.",
	"source_map_incompatible":   "The retained source mapping does not match the selected stream, channels, sample window or processing policy. Inspect the recording and elect compatible reuse or new diarization.",
	"backup_checksum":           "The backup bytes do not match their recorded checksum. Restore an intact backup or reacquire the damaged object.",
	"backup_size":               "The backup object size does not match its recorded size. Select a complete intact backup.",
	"backup_identity":           "The backup object identity does not match its inventory. Select an intact backup for the current restore plan.",
	"model_not_acquired":        "The model is registered but its bytes are not acquired. Use models acquire with the original manifest file, or inspect/retry its existing acquisition work.",
	"model_damaged":             "Verified model bytes do not match their manifest. Reacquire the exact pinned bundle before using it.",
	"model_storage_unavailable": "Model storage is unavailable. Check the selected artifact backend before retrying verification.",
	"profile_prerequisite":      "The operating-system user profile directory is unavailable. Restore the required user profile environment before runtime startup.",
	"runtime_directory":         "The runtime control directory is unavailable or cannot be secured for this user. Check ownership and access permissions.",
	"runtime_owner":             "Another runtime owns this workspace. Use the live owner or wait for it to finish before exclusive maintenance.",
	"runtime_endpoint":          "The owned runtime endpoint could not be opened. Check runtime directory permissions and stale ownership.",
	"runtime_catalog":           "Runtime application or catalog startup failed. Inspect workspace configuration and catalog access; startup does not prove provider readiness.",
	"startup_launch":            "The runtime executable could not start. Check the installed package, executable path and platform prerequisites.",
	"startup_exit":              "The runtime exited before reporting readiness. Check the installed package and workspace prerequisites.",
	"startup_report":            "The runtime returned an invalid startup report. Use a matching complete package and retry.",
	"startup_timeout":           "Runtime readiness exceeded the startup deadline. Inspect workspace/runtime prerequisites before retrying.",
	"wait_timeout":              "The selected wait limit elapsed; durable work continues. Use work show, work wait or work cancel with the same work ID.",
	"wait_interrupted":          "Waiting stopped; durable work was not cancelled. Inspect or wait for the same work ID.",
}

// Condition retains only an allowlisted locally generated reason.
func Condition(code, reason string) *Error {
	e := Fail(code)
	if message, ok := operatorReasons[reason]; ok {
		e.Reason = reason
		e.Message = message
	}
	return e
}

func SafeError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return Condition(e.Code, e.Reason)
	}
	return Fail("operation_failed")
}

func OperatorReasons() []string {
	out := make([]string, 0, len(operatorReasons))
	for k := range operatorReasons {
		out = append(out, k)
	}
	return out
}
