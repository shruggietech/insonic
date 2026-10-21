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

## Implemented recovery boundary (2026-10-21)

The binding retains its existing architecture-specific SDK implementations behind synchronous error-returning facades. First failure wins across synchronous loader/version/bootstrap calls, callbacks and process-failure notifications during startup/configuration. Resource release waits for both callbacks and captured native frames to unwind, including owner destruction inside a native call. The existing bool Embed API remains compatible; Wails uses EmbedWithError and scoped setup errors. Runtime process-failure policy outside startup remains unchanged.

The maintained Wails snapshot contains all 956 checksum-bound upstream files. Exactly app_dev.go, app_production.go and the Windows frontend change; licensing/patch notices and fake frontend lifecycle tests are the three additions. Linux platform source remains unchanged. The common application-loop guard applies to failed frontend startup on both platforms.

The real Windows fixture adds one expected-failure child before the first of the unchanged twelve successful cases. Its case directory grants inheritable current-user Modify. An exclusive owned regular file obstructs the intended profile; the failed child must return a typed allowed native stage, save a matching generated private event, release its owned resources/apartment and exit normally. The parent independently requires natural owned-descendant release, verifies blocker identity and bytes, then removes only that blocker. The next existing normal child uses the same intended profile. No unchanged failure is retried. Eighteen-second child and hundred-second matrix bounds remain unchanged.

The audited binding selector retains all five existing CI tests and adds every TestStartup test. Upstream TestCookieManager is excluded because it creates a visible window; real browser exercise remains the explicit unattended native/package paths.
