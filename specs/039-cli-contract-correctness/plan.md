# Implementation Plan: CLI contract and operator-feedback correctness

**Branch**: `codex/039-cli-contract-correctness` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

## Summary

Close #116, #117, #118, #119, #120, #121, #131, #132, #133, #134 and #144 through shared command guidance, validation and result contracts. Extend existing installed CLI qualification. #114 and #147 retain their broader open scope. The dated owner-approved extension adds #154 native startup/Exit repair as the twelfth target.

## Technical Context

- Language/version: Go 1.27.1, Node >=22.12, Python 3.12; existing React/TypeScript and Next.js.
- Dependencies: Existing JSON Schema validator, catalog adapters, runtime IPC and ProcessTree qualification. No new product dependency.
- Storage: SQLite/PostgreSQL catalogs, filesystem/S3 artifacts and LadybugDB/ArcadeDB adapters remain supported. Forward catalog migration widens duration checks.
- Testing: Go ordinary/native/race and vet, repository/schema Node tests, Python maintainer/installed fixtures, frontend/site checks/builds and existing Windows/Linux/backend CI.
- Target: Windows/Linux x86-64 CLI and shared-runtime desktop consumers.
- Constraints: Existing 128-line/64-item/6-depth/512-byte-value human bounds; complete elected CI <=660 seconds. Exact integers, private input exclusion and hidden owned Windows launches remain mandatory.
- Scope: All accepted commands/groups/options and generated examples, eleven CLI defect closures plus the owner-approved native repair, additive generated diagnostics and compatible catalog evolution.

## Constitution Check

Pre-research and post-design gates pass. Approved scope is retained; no new usage policy or attended verification gate. Shared core/schema corrections align GUI consumers. Immutable identity bytes, historical migration digests and release records remain intact. Public docs state contracts; slice evidence stays internal. Push/PR is authorized; merge/release/site publication is outside this slice.

## Project Structure

- `cmd/insonic/`: registry/help metadata, parsers, main/setup, human renderer and focused tests.
- `internal/contracts/`, `internal/catalog/`, `internal/models/`, `internal/app/`: generated failures, duration migration/text semantics and observed results.
- `schemas/`: shared validation, help templates and current request/response/snapshot contracts.
- `scripts/cli_contract_qualification.py` and existing media/query/documented-workflow fixtures: semantic example execution and explicit coverage.
- `docs/v1.0.1/`, `CHANGELOG.md`: current interface and compatibility contracts.
- This feature: spec, plan, research, data-model, contracts/operator-contract, quickstart, tasks, checklists and verification.

## Implementation

1. Establish metadata/regression contracts before behavior changes. Preserve or deliberately update root-help enumeration consumers.
2. Parallel disjoint work: command help/examples; observed results/human summaries; duration migration/Unicode validators. Parent owns validation diagnostics, export classification and installed integration. Serialize shared main/schema edits.
3. Extract actual emitted examples, substitute only documented current identities/revisions/paths/fixtures, prepare owned prerequisites and assert semantic outcomes. Parser/schema-only success cannot close #119.
4. Run focused then complete required checks and installed fixtures, commit, push and publish a non-draft PR. Observe exact-head CI and at most two review rounds; resolve every thread and stop unmerged.

## Catalog Evolution

Preserve exact schema-9 DDL/digest, then introduce schema 10 with 1..600000 duration checks. SQLite rebuilds only the job table in the existing exclusive transaction, copies every column and checks foreign keys. PostgreSQL replaces the named constraint. Retain schema-9 snapshot acceptance and test populated old catalogs, IDs/attempts/receipts, replay/restore and new boundaries. Historical DDL remains unchanged.

## Verification and Delivery

Regression-first checks cover each issue and every generated example, privacy and exact numbers. Run npm check/test, Go/vet/native and affected backend tests, Python suite, frontend/site check/test/build and existing full package CI. Use hidden noninteractive launchers; no owner GUI or installer. CI caching/parallelism may improve efficiency without weakening gates. Initial automatic review is round one; one optional @Codex request is round two. Handoff reports exact head, CI, direct thread audit, tracking state and unverified behavior.

## Complexity Tracking

No constitution exception or new framework. Forward catalog migration is a demonstrated necessity for approved ten-minute duration support.

## 2026-10-19: owner-approved native startup extension

Add #154 as the twelfth issue. Inspect exact WebView2 HRESULTs, startup/DOM checkpoints, profile ownership and Go/Python Win32 job boundaries before selecting a correction. Reuse the bounded disposable incident diagnostic; its observations confer no delivery authority. Isolate the demonstrated lifecycle with a regression or controlled before/after experiment, then correct only the necessary application/process/qualification boundary. Do not assume that shared profiles or resource pressure caused the failure.

Potential touch-points are cmd/insonic-desktop/application.go, exit_qualification.go and platform helpers; internal/process/capture.go and tree_windows.go only if their owned-descendant contract is implicated; scripts/diagnose-desktop-startup.py, package-desktop.py and their existing workflow/tests. Keep production defaults unless a demonstrated necessity requires a change. Use fixture-owned profiles and observe owned descendant release if profile isolation is selected. Preserve actual native actions, playback cleanup, durable-work/session survival, all delivery scopes and exact-byte/source binding.

Run focused regression-first checks, required root/Go/vet/native/race/Python/frontend/site checks as affected, repeated real Windows cold/warm native qualification and complete exact-head CI <=660 seconds. Existing external review rounds remain two of two; do not request another. Record post-merge-main as unverified until the owner merges.

### Native mnemonic routing refinement

The real SendInput probe demonstrates that the pinned Wails Windows accelerator callback drops the system-key kind when forwarding WebView accelerators. Add a Windows-only owned-UI-thread message adapter before the common Wails Run call: translate only an actual Alt+F system key for an owned menu-bearing root into the normal system-menu message. Unrelated messages pass through the hook chain; no global shortcut or other-process hook is registered. Keep its callback alive and unhook/unlock on every normal/error return. Linux retains its native route. Test the Win32 key-message contract, ABI, invalid ownership and installation/cleanup without launching UI; real CI proves the actual focused-WebView route. This refines the already approved native menu contract without a new dependency or broader scope.

### Observed message boundary and final mnemonic design

The bounded 47c9b9f trace proves that the UI-thread hook is too late: the hook and native root have the same thread, but Alt and F arrive as ordinary WM_KEYDOWN with flags=0. Remove that ineffective hook. The Windows native bridge instead exposes the owned File-menu opener, and the focused document routes an actual Alt+F KeyboardEvent with its intact Alt modifier through it. Bind both the application document and offline-help document, preserve ordinary/control/meta/repeat inputs, report opening failures, and remove listeners on teardown. Windows qualification waits for the actual listener-ready acknowledgment within the existing watchdog/caller bounds before sending the real chord; it still observes native menu ownership, sends real X, requires the real callback and normal host/browser cleanup, and checks durable-work/playback survival. Linux has no Windows menu binding and retains its native handling.

### Complete automated WebView ownership

Concurrent load reproduced 800700aa in the historical parent and the current help-only predecessor, before the isolated Exit cases. Extend the same ownership contract to every automated native WebView: Python supervisor owns a fresh per-probe profile and waits for normal job completion under the original help/full-WebView child deadline; Go consumes that path only for explicit qualification modes. Preserve normal product profile defaults. A failed/unknown release retains its owned profile for diagnosis, never deletes it while ownership is unresolved and never confers success. Historical failures in the dedicated before/after diagnostic remain recorded baseline evidence; only current failures invalidate current observation completion. This diagnostic distinction cannot confer delivery authority or weaken any package gate.

### Native integer-pointer lifetime

The real menu regression demonstrates stale stack addresses across the generic uintptr native wrapper. Preserve pointer-valued arguments through the wrapper using the Go compiler lifetime directive. Apply the same contract to generated job-information APIs, and explicitly pin typed MSI input/output pointers before converting them to uintptr. Retain actual Windows menu/job/MSI regressions and existing failure/timeout proofs. Preserve MSI inspection errors separately from actual identity mismatches. This is a proportional shared native-boundary correction; no dependency or security setting changes.

### Final native readiness and failure ownership

Retain failed Exit profiles and workspace authority after runtime join; removal belongs to normal successful qualification only. The native selection probe requires both general menu mode and an active owned popup before real X input, excluding system-menu mode, and retains the observed flags/focus/owner. This distinguishes menu-bar activation from popup readiness without a sleep, deadline extension or callback substitution. See the Windows GUITHREADINFO contract at https://learn.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-guithreadinfo.

### Explicit WebView controller shutdown

Run 39252465027 proves a normal zero-exit host can leave an owned WebView process running at the original deadline. The pinned Wails WindowClose destroys its native window without calling the controller Close operation; the pinned binding has no public close implementation or parent-window lifetime hook. Microsoft documents Close as releasing WebView/browser resources and breaking event-handler cycles. Keep the pinned binding as a repository-maintained source patch: attach an owned UI-thread window subclass and close its controller once during destruction, preserving the normal window procedure. Preserve exact upstream module h1/per-file hashes and a separate patched-tree h1, root verification, original licenses and explicit patch notice. Allow only this exact repository-local module replacement through packaging provenance, bound to the clean source revision and verified manifest; all unrelated local replacements remain rejected. Add a real unshown native-window/fake-COM lifetime test and repeat real packaged browser-release proofs. This is a proportional correction to the demonstrated shutdown boundary, without a Wails fork, SDK version upgrade or timeout increase.

### Joined fixture activation before keyboard input

The native Exit probe called Wails WindowShow asynchronously after DOM readiness immediately before input; the pinned implementation later shows, foregrounds and focuses the root through its UI dispatch queue. An already visible root did not establish that this queued operation had completed. Remove that unjoined focus mutation. Only the disposable Exit startup now locates its owned root and completes native ShowWindow/foreground activation before closing readiness; the existing watchdog covers preparation. Two synchronous show calls consume the launcher's first-show hide flag and then establish visibility. No owner GUI is launched locally. Native qualification still requires foreground ownership, actual keys, actual callbacks and natural release. Remove the temporary same-thread trace and repeat the uninstrumented full/load gates. This fixes a demonstrated source ordering hazard; prior intermittent selection and pre-DOM signatures remain distinct observations, not proof of one private WebView cause.

### Complete framework activation, joined on its owning UI thread

The native-only ShowWindow variant at ffc348d does not preserve Wails' controller focus and failed real Alt+F/Alt+F4 input. Keep the framework's complete ShowWindow behavior. A short-lived WH_GETMESSAGE dispatcher on only the owned UI thread responds to one unique owned WM_NULL marker; it executes the two Wails activation calls on that thread, where Wails Invoke runs inline. Completion is observed before readiness, and successful hook removal is required before any input. Messages are passed through unchanged. A two-second operation bound fits inside the unchanged startup/watchdog/child limits; timeout is failure. An unshown real-window regression checks absent UI dispatch, actual UI-thread ownership and completed native work, with a controlled old unjoined-boundary comparison. No queue/key tracing or hook remains during real menu input.

Wails navigation completion hides/shows its controller after starting OnDomReady. Apply the joined full-framework activation again after frontend readiness, immediately before input, so the startup activation and final input focus are both established under the same unchanged watchdog. The short-lived dispatcher is removed after each activation and is absent during actual keys.

### Explicit owned focus transition

9143780 passes nine full production parents (108 real Exit children) under concurrent load before a new cold menu misses the document mnemonic. Joined activation alone does not guarantee a fresh WM_SETFOCUS if the native root already owns focus after navigation. The pinned framework forwards controller focus only from that event. Before Wails activation, clear focus on the fixture's own UI thread, then perform the existing joined show/focus calls and retain bounded before/after/root handles. This creates the required event and does not modify any other thread or owner window. All original limits/actions remain.

### Complete COM ownership after controller Close

e778f51 passes twelve full source parents (144 natural browser releases), yet its exact-byte archive observes a zero-exit menu host with one successful controller Close and three actual owned WebView processes still running at the original deadline. The binding retains its controller/environment references, adds a redundant reference after GetCoreWebView2, leaks its controller3 query reference and never balances its existing pinned-main-thread CoInitializeEx. Follow the documented Close plus interface-release pattern. Release owned WebView/controller/environment references on window destruction, remove the duplicate getter AddRef, release the temporary queried interface, then finalize the successfully initialized COM apartment on that original thread after Wails returns. Other normal desktop returns finalize the same initialization idempotently. Preserve primary Close/error exits; refuse another thread or still-owned windows. Native apartment tests use real CoGetApartmentType and owned worker initialization, without showing UI. Extend the exact maintained patch allowlist to the two implicated upstream files, retaining every original byte hash and all four additions. No process termination or larger deadline can confer release.
