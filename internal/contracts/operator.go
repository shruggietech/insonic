// SPDX-License-Identifier: Apache-2.0
package contracts

import "errors"

var operatorReasons = map[string]string{
	"json_empty":                     "The JSON input is empty. Supply one JSON object using this command's contextual --help template.",
	"json_syntax":                    "The JSON syntax is incomplete or invalid. Correct the reported byte position and supply exactly one JSON document.",
	"json_encoding":                  "The JSON input is not valid UTF-8. Save the file as UTF-8 and retry.",
	"json_duplicate_key":             "The JSON input repeats an object member. Keep each member name exactly once at every level.",
	"json_extra_document":            "The input contains more than one JSON document. Supply exactly one complete document.",
	"input_shape":                    "The JSON input has the wrong shape. Use the object or collection shown by contextual --help.",
	"input_unknown_field":            "The JSON input contains an unsupported member. Remove members absent from this command's input contract.",
	"input_required":                 "A required input field is missing. Supply the indicated field using contextual --help.",
	"input_type":                     "An input field has the wrong type. Use the indicated JSON type and contextual --help.",
	"input_enum":                     "An input field has an unsupported choice. Select one of the values listed in contextual --help.",
	"input_range":                    "An input number is outside its supported interval. Use the indicated bound and contextual --help.",
	"input_length":                   "An input string or collection exceeds or falls below its supported length. Use the indicated constraint.",
	"input_version":                  "The input contract version is unsupported. Use a packaged supported version and contextual --help.",
	"input_constraint":               "An input field does not meet its contract. Correct the indicated field using contextual --help.",
	"input_bytes":                    "The input exceeds its UTF-8 byte budget. Supply a smaller input within the indicated maxBytes.",
	"credential_value_limit":         "A protected credential value or passphrase must contain 1..2048 UTF-8 bytes. Supply it through protected stdin; no value was returned.",
	"credential_input_shape":         "Protected input must be one object containing only value, passphrase or session as required by the selected backend. Use credentials help for the protected template.",
	"output_exists":                  "The selected output destination already exists. Choose a new path; this operation does not overwrite existing files.",
	"output_parent_missing":          "The output parent directory does not exist. Create it or choose an existing writable directory and retry.",
	"output_permission":              "The output destination cannot be written with the current permissions. Choose a writable directory or correct its access permissions.",
	"output_directory":               "The output operand names a directory. Choose a new file path inside a writable directory.",
	"output_space":                   "The output could not be written because the destination has insufficient space. Free space or choose another destination and retry.",
	"output_write":                   "The output could not be created, written, synchronized or closed. Check the destination and filesystem before retrying.",
	"documentation_open":             "Installed offline documentation could not be opened. Open help/index.html in the program directory with your HTML browser, or use installer Repair to restore matching help files.",
	"provider_auth":                  "The provider rejected authentication. Check the elected credential or server account.",
	"provider_missing":               "The elected provider endpoint or model is unavailable. Verify the endpoint, model and adapter protocol.",
	"provider_busy":                  "The elected provider is busy or rate limited. Retry the same selection later.",
	"provider_redirect":              "The provider returned a redirect. Select the intended endpoint explicitly.",
	"provider_unavailable":           "The elected provider could not be reached. Check its service, endpoint and TLS.",
	"provider_failure":               "The elected provider rejected the request. Check its status and protocol before retrying.",
	"provider_protocol":              "The provider returned an incompatible or malformed response. Check its adapter protocol and model output.",
	"provider_timeout":               "The provider deadline expired. Retry the elected service or increase its timeout within supported bounds.",
	"provider_response_limit":        "The provider response exceeded its byte or metadata bound. Request smaller output, or raise the configurable response limit within supported bounds; fixed metadata bounds cannot be raised.",
	"native_result_limit":            "The native query exceeded the 500-row or 512 KiB result bound. Narrow the query or return fewer or smaller values.",
	"artifact_materialization_limit": "The artifact exceeds the elected materialization max_bytes. Inspect its size and elect a sufficient max_bytes within supported bounds.",
	"native_statement":               "Submit one read statement with at most one trailing semicolon followed only by whitespace or comments. Split multiple statements into separate query native invocations.",
	"native_function":                "A function call is outside the supported pure-function subset. Use COUNT, SUM, MIN, MAX, AVG, COALESCE, TOSTRING, TOINTEGER, LOWER, UPPER, ABS, SIZE, LENGTH or COLLECT where supported by the selected dialect; see query native --help. Calls such as range are not supported.",
	"native_syntax":                  "The native read has empty or malformed text, an unterminated quote/comment or an invalid token. Correct the statement and consult query native --help for the supported grammar.",
	"native_read_only":               "Native queries require a read statement: SELECT for SQL, or MATCH, OPTIONAL MATCH, RETURN, WITH or UNWIND for Cypher, optionally prefixed by EXPLAIN. Remove mutations, transaction control, procedures and profiling parameters; use explicit maintenance operations for effects.",
	"backend_pending_plan":           "This directory cannot reconcile this provisioning input. Reuse its retained provision-intent configuration and credential backend; changed input, a missing receipt or a changed credential selection conflicts.",
	"backend_s3_bucket":              "The selected S3 bucket access could not be established. Verify the endpoint, existing bucket, credentials and TLS; provisioning does not create S3 buckets.",
	"backend_plan":                   "The backend plan is invalid. Correct the reported fields and use workspace plan --help for complete supported shapes.",
	"backend_local_transport":        "PostgreSQL local requires a literal loopback address, pinned localhost or Unix socket. Use verify-full or explicitly named insecure for a remote host.",
	"backend_endpoint_tls":           "Credential-bearing connected ArcadeDB endpoints require HTTPS; verify the endpoint and certificate.",
	"ollama_endpoint":                "Choose an HTTP(S) Ollama service root without credentials, query strings or fragments.",
	"ollama_unavailable":             "Ollama is unreachable. Start the elected service or verify its endpoint and TLS.",
	"ollama_provider":                "The elected Ollama provider failed. Check its status and retry without changing routes.",
	"ollama_auth":                    "Ollama rejected authentication. Check the elected credential or server account.",
	"ollama_missing":                 "The selected model or native Ollama route is unavailable. Refresh model metadata and verify the service protocol.",
	"ollama_busy":                    "Ollama is busy or rate limited. Retry the same elected selection later.",
	"ollama_redirect":                "Ollama returned a redirect. Select the intended service root explicitly.",
	"ollama_protocol":                "Ollama returned an incompatible or malformed response. Check the service/model and native adapter support.",
	"ollama_capability":              "The elected model does not advertise text completion. Select a task-compatible model.",
	"ollama_route":                   "This selection cannot prove local inference. Inspect its model route and explicitly elect hosted inference if intended.",
	"workspace_missing":              "No workspace was found. Initialize a library or select its directory with --workspace.",
	"input_missing":                  "The selected input file is unavailable. Check its path and permissions.",
	"input_invalid":                  "The selected input does not match this command's contract. Use contextual --help for the required fields.",
	"revision_stale":                 "The selected revision is no longer current. Inspect the item and retry with its current revision.",
	"credential_duplicate":           "This credential ID is already stored. Inspect its status or use credentials replace with the same ID.",
	"credential_capacity":            "The credential store is full. Remove an unused credential before adding another.",
	"credential_locked":              "The saved credential is locked in this manager. Unlock the vault through protected input.",
	"credential_bootstrap":           "Credential bootstrap failed. Check the selected backend, vault file and protected passphrase input; no value was logged.",
	"source_map_incompatible":        "The retained source mapping does not match the selected stream, channels, sample window or processing policy. Inspect the recording and elect compatible reuse or new diarization.",
	"backup_checksum":                "The backup bytes do not match their recorded checksum. Restore an intact backup or reacquire the damaged object.",
	"backup_size":                    "The backup object size does not match its recorded size. Select a complete intact backup.",
	"backup_identity":                "The backup object identity does not match its inventory. Select an intact backup for the current restore plan.",
	"model_not_acquired":             "The model is registered but its bytes are not acquired. Use models acquire with the original manifest file, or inspect/retry its existing acquisition work.",
	"model_damaged":                  "Verified model bytes do not match their manifest. Reacquire the exact pinned bundle before using it.",
	"model_storage_unavailable":      "Model storage is unavailable. Check the selected artifact backend before retrying verification.",
	"model_access_required":          "This gated model download needs a locally resolvable protected read-token reference. Register, unlock or repair the reference, or reuse complete verified local bytes. Preview does not check upstream authentication.",
	"profile_prerequisite":           "The operating-system user profile directory is unavailable. Restore the required user profile environment before runtime startup.",
	"runtime_directory":              "The runtime control directory is unavailable or cannot be secured for this user. Check ownership and access permissions.",
	"runtime_owner":                  "Another runtime owns this workspace. Use the live owner or wait for it to finish before exclusive maintenance.",
	"runtime_endpoint":               "The owned runtime endpoint could not be opened. Check runtime directory permissions and stale ownership.",
	"runtime_catalog":                "Runtime application or catalog startup failed. Inspect workspace configuration and catalog access; startup does not prove provider readiness.",
	"startup_launch":                 "The runtime executable could not start. Check the installed package, executable path and platform prerequisites.",
	"startup_exit":                   "The runtime exited before reporting readiness. Check the installed package and workspace prerequisites.",
	"startup_report":                 "The runtime returned an invalid startup report. Use a matching complete package and retry.",
	"startup_timeout":                "Runtime readiness exceeded the startup deadline. Inspect workspace/runtime prerequisites before retrying.",
	"wait_timeout":                   "The selected wait limit elapsed. Inspect the work ID with work show, work wait or work cancel; this wait result does not establish its current state.",
	"wait_interrupted":               "Waiting stopped; durable work was not cancelled. Inspect or wait for the same work ID.",
}

// Condition retains only an allowlisted locally generated reason.
func Condition(code, reason string) *Error {
	e := Fail(code)
	if rule, ok := assetReasons[reason]; ok && rule.code == e.Code {
		e.Reason = reason
		e.Message = "An elected local asset " + rule.message
		return e
	}
	if message, ok := operatorReasons[reason]; ok {
		e.Reason = reason
		e.Message = message
	}
	return e
}

func SafeError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		if rule, ok := assetReasons[e.Reason]; ok && rule.code == e.Code && e.Asset != nil {
			return AssetFailure(e.Reason, e.Asset.Role, e.Asset.Name)
		}
		if e.Validation != nil && inputReason(e.Reason) {
			return InputFailure(e.Reason, *e.Validation)
		}
		return Condition(e.Code, e.Reason)
	}
	return Fail("operation_failed")
}

func OperatorReasons() []string {
	out := make([]string, 0, len(operatorReasons))
	for k := range operatorReasons {
		out = append(out, k)
	}
	for k := range assetReasons {
		out = append(out, k)
	}
	return out
}
