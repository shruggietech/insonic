# Implementation Plan: Windows desktop startup and Exit reliability

**Branch**: `codex/038-windows-webview-startup` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

## Summary

Deliver the owner-approved diagnostic checkpoint for #154: preserve native failure evidence, validate bounded historical/source observations on disposable Windows, retain the primary failed delivery producer and qualify the complete checkpoint. Keep #154 open for the unreproduced startup cause and its supported correction.

## Technical Context

**Language/Version**: Go 1.27.1, Python 3.12+, Node 22+.

**Primary Dependencies**: Wails 2.16.0, go-webview2 1.0.22, Windows WebView2 Runtime, existing process/job and package/inventory tooling.

**Storage**: Fixture-owned temporary evidence/profile data; no catalog schema or operator profile migration.

**Testing**: Go child-result/native tests, Python qualification/pipeline tests, Node workflow/artifact tests, required root and affected site checks.

**Target Platform**: Windows x86-64 diagnosis; Linux compatibility and complete delivery matrix.

**Project Type**: CLI-first application with Wails desktop and governed package CI.

**Performance Goals**: Focused diagnostic before full matrix; final elected CI at most 660 seconds.

**Constraints**: No local GUI/MSI launch, blanket process termination, suppressed native action, blind retry or weakened publication proof. At most two requested review rounds; no merge/release.

**Scale/Scope**: One issue, bounded native launch sequence and delivery diagnostics.

## Constitution Check

Before and after design: pass. Preserve owner scope, shared runtime/startup/Exit, isolated fixtures, exact-source delivery and 660 seconds. Public docs remain outcome-focused. No exception or new owner gate.

## Phase 0: Research

Two read-only research agents independently confirmed stdout loss and shared-profile/default job teardown while rejecting a proven profile-collision claim. [Research](research.md) records evidence and alternatives.

## Phase 1: Design

1. Extract child-result interpretation from runExitGUIChild; preserve both streams and result metadata. A synthetic failed child proves stdout retention before hosted diagnosis.
2. Add a focused option to the existing manual Windows diagnostic workflow. Restore exact incident assets with source/size/hash/inventory checks. Observe original parent behavior separately from direct native-child probes where old parent bytes discard stdout. Diagnostics confer no publication authority.
3. Record bounded reproduction outcomes without promoting hypotheses into a startup fix. The owner deferred actual HRESULT/cause recovery and correction under open #154.
4. Preserve complete existing native action qualification and use repeated diagnostic observations. Separate historical/source GUIs, exact historical source/compiler authority, bounded readers and exact compiler environments form the implemented diagnostic boundary. New cold/warm profile lifecycle qualification is deferred under #154.
5. Persist bounded failed-producer evidence before the exact-byte join aborts, without accepting failed/missing/foreign proof or fabricating completed installer receipts.

See [data model](data-model.md), [internal contracts](contracts/native-diagnostics.md) and [validation guide](quickstart.md). No public runtime API change is planned.

## Project Structure

- cmd/insonic-desktop/exit_qualification*.go and tests: failure evidence, demonstrated correction and native action coverage.
- internal/process/: existing bounded capture/job ownership, changed only if demonstrated cause requires it.
- scripts/diagnose-desktop-startup.py and tests/desktop_startup_diagnostic_test.py: incident-bound focused probe.
- .github/workflows/installer-invocation-diagnostic.yml: explicit diagnostic election.
- scripts/wait-ci-artifact.mjs, tests/ci-artifact.test.mjs, .github/workflows/windows-desktop-qualification.yml: primary producer-failure evidence.
- CHANGELOG.md and docs/v1.0.1/development.md: changed behavior and diagnostic ownership.

## Decisions and Boundaries (2026-10-19)

Prefer error-preserving diagnosis to speculative profile changes. Reuse the manual disposable workflow rather than repeatedly running the full matrix. Old-byte direct-child observations are diagnostic only; exact newly built production qualification remains required. Push/PR and two review rounds are authorized. Stop unmerged; actual main confirmation follows owner merge. Native cause investigation is buildable work, not an unresolved owner requirement.

## Complexity Tracking

No constitution violations or new runtime framework.

## Owner-approved checkpoint scope (2026-10-19)

The owner explicitly accepted a diagnostic checkpoint with #154 left open. No actual startup cause or correction has been demonstrated. The initial high convergence findings for original cause repair/cold-warm lifecycle work are deferred issue work, not checkpoint completion claims. Complete root/affected tests, exact-head ZIP/MSI/setup/recovery qualification within 660 seconds, at most two review rounds and the unmerged owner handoff remain required.
