# Design decisions: Startup diagnostics and recovery

## Returned error and feedback

Decision: Wrap the returned playback/window/close stages without losing the original error chain. Use native platform feedback after cleanup, with an injected presentation sink for unattended tests.

Rationale: main currently discards the returned cause; a browser-dependent dialog cannot report browser initialization failure. Preserve the ordinary persistent profile and suppress interactive presentation in explicit qualification paths.

Alternatives: Stderr-only reporting is inadequate for consoleless launch. A new telemetry service or framework replacement is outside scope.

## Private diagnostics and Windows access

Decision: Reuse the bounded private logger, allowlisting generated startup stage and numeric system error. Storage/presentation failure supplements feedback but never replaces its primary cause. Request WRITE_OWNER only when a genuinely new file needs ownership normalization.

Rationale: private_windows.go requests WRITE_OWNER and sets ownership unconditionally for new files, including files already owned by the caller in inherited-Modify directories. Preserve original-handle equality, reparse rejection, existing-owner verification and restrictive protected DACLs.

Alternatives: Broad ACLs, elevation and disabling owner checks are rejected. Requiring every new file to have the caller's initial owner would narrow the existing legitimate owner-normalization behavior unnecessarily.

Reference: [Microsoft file security/access rights](https://learn.microsoft.com/en-us/windows/win32/fileio/file-security-and-access-rights).

## Startup completion and framework boundary

Decision: In the second increment, explicitly complete binding initialization with success or first failure. Add error-returning embedding, return each relevant synchronous/callback error, and skip ordinary loop entry after frontend failure.

Rationale: Spec Kit's bounded read-only research traced binding errorCallback/os.Exit, EnvironmentCompleted(nil) leaving the startup pump pending, Wails setupChromium ignoring Embed and using log.Fatal, and dev/production App.Run entering the loop even after frontend error. Correct only these demonstrated boundaries.

Maintain the exact Wails 2.16.0 baseline checksum `h1:hLYrSHUUq6hRXl3rWT9ZXiyFIgM94nVN9Ns6iEJzxyw=`. Existing maintained-module source/notice enforcement must expand to exactly the two approved modules, retaining arbitrary replacement refusal. Do not edit module-cache bytes.

Cleanup reuses existing owned-window teardown after callback unwinding. Balance the caller-owned settings reference in the touched framework path. A subsequent fresh process may launch normally; no automatic retries or persistent-profile deletion.

Reference: [WebView2 owning-thread/callback rules](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/threading-model) and [Go wrapped errors](https://go.dev/blog/go1.13-errors).

## Verification boundary

Use fake/injected failures and unshown windows locally. Existing unattended Windows CI supplies real browser/package acceptance and controlled failure/relaunch coverage when the recovery increment is ready. Preserve all existing operation limits and cases. No attended desktop exercise or common historical root-cause assertion.
