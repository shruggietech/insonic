# Implementation Plan: Desktop startup diagnostics and recovery

**Branch**: `codex/042-startup-diagnostics` | **Date**: 2026-10-21 | **Spec**: [spec.md](spec.md)

## Summary

Deliver two independently mergeable increments. First preserve returned startup causes, present native failure feedback without relying on a browser or console, retain bounded whitelisted private diagnostic events, and remove unnecessary Windows ownership-changing rights. Then correct demonstrated synchronous/asynchronous initialization outcome and owning-thread cleanup defects through the existing maintained binding and a narrow pinned-framework patch.

## Technical Context

**Language/Version**: Go 1.27.1, existing Windows APIs and Linux GTK, Node/Python source and qualification tooling.
**Primary Dependencies**: Existing Wails 2.16.0, maintained go-webview2 1.0.23, golang.org/x/sys and private diagnostic logger. No upgrades.
**Storage**: Existing bounded private JSONL logger and lock/rotation policy. Desktop startup events use the per-user configuration directory under insonic/diagnostics.
**Testing**: Injected returned errors and presentation/storage sinks; real Windows ACL/file-identity tests; fake COM and unshown message-only windows; existing unattended native/package CI.
**Target Platform**: Windows and Linux desktop; Windows private-file and native initialization correction.
**Project Type**: Existing CLI/shared-runtime and desktop wrapper.
**Performance Goals**: Preserve native operation allowances; 660-second advisory qualification target, 900-second hard limit after readiness, existing 15-second provisional reserve.
**Constraints**: No attended owner-desktop exercise, retry loops, profile deletion, forced browser termination, release, unrelated dependency merge or discretionary hosted review gate.
**Scale/Scope**: Returned startup errors, private diagnostic-file creation and concrete initialization termination/cleanup only. #154 historical causes remain distinct.

## Constitution Check

Constitution 4.0.0 governs. CLI diagnostic opening remains fail-before-command-effects. Native feedback improves the GUI wrapper without new core GUI-only operations. Diagnostic records preserve privacy through fixed metadata rather than arbitrary error strings. Exact upstream source/notice authority and every elected product/integrity check remain mandatory. Existing publication and squash-merge authorization covers both increments. Pre-research and post-design checks pass; no new owner decision is required.

## Project Structure

### Documentation

This directory contains spec.md, plan.md, research.md, data-model.md, contracts/startup-failure.md, quickstart.md, tasks.md, requirements checklist and verification.md.

### Source

- cmd/insonic-desktop/main.go and application.go: wrapped returned causes, user reporting and cleanup test seam.
- cmd/insonic-desktop/startup_failure*.go: generated failure view, default private event persistence and native platform feedback.
- internal/diagnostics/logger.go and private_windows.go: allowlisted startup metadata, actionable opening boundaries and least-required rights while preserving identity/owner/reparse checks.
- third_party/go-webview2/pkg/edge: explicit startup outcome/error propagation and focused fake-COM/message-only regressions (second increment).
- third_party/wails: complete verified pinned baseline with a narrow behavioral patch (second increment only).
- scripts/webview-source.mjs, scripts/check.mjs, scripts/go_module_notices.py and qualification metadata/tests: extend the existing exact maintained-module authority to the required pinned Wails patch, keeping arbitrary replacements rejected.
- docs/v1.0.1/desktop.md, cli.md, development.md and CHANGELOG.md: authoritative behavior and dated decisions.

## Chronological execution

1. Specify outcomes and resolve privacy, native feedback and fresh-process recovery assumptions. Reconcile S041 main acceptance from existing external evidence.
2. Design and analyze both increments; keep the first increment independently useful.
3. Add failing returned-error, denied-diagnostic, private-ACL/path/ownership and metadata privacy regressions.
4. Implement returned-error wrapping/reporting and diagnostic-file correction; use existing platform APIs for native feedback (Windows MessageBox and existing Linux GTK dependencies). Qualification paths remain noninteractive. A storage or feedback error never replaces the original failure.
5. Run required root, affected Go, site and encoding/diff checks; publish one frozen first-increment PR, qualify, merge and verify exact main.
6. Add focused failing startup-outcome/cleanup regressions, then apply the smallest binding and pinned-framework correction. Preserve Linux/dev success behavior and source authority.
7. Run focused/native/package acceptance, publish and merge the second frozen increment with passing required checks, then verify exact main. Close only independently satisfied issue criteria.

## Decisions and alternatives (2026-10-21)

- Preserve original error chains with generated stages; expose actionable failure information through native feedback after application cleanup. Browser dialogs cannot cover browser initialization failure, and stderr alone cannot cover ordinary GUI launch.
- Persist only existing generated event fields plus an allowlisted failure stage and optional numeric system error. Never persist arbitrary error text or paths. Retain the original error for user feedback if private storage fails.
- Reopen private Windows files initially with READ_CONTROL, WRITE_DAC and FILE_READ_ATTRIBUTES. Request WRITE_OWNER and set ownership only for a genuinely new file whose observed owner differs; revalidate original handle identity/reparse state for any replacement handle. Existing foreign files remain refused. This preserves legitimate new-file owner normalization while fixing ordinary inherited-Modify creation.
- Distinguish creation/path, privacy and lock-contention opening failures using generated corrective messages, preserving unavailable classification and fail-before-effects.
- Read-only Spec Kit research confirms Wails privately creates Chromium, ignores Embed's result and can enter the application loop despite frontend error. Existing callbacks cannot make this path return cleanly. A narrow maintained Wails patch is necessary for the second increment.
- Go replacement is module-wide. Preserve a complete verified Wails 2.16.0 baseline (956 files, 8,705,515 bytes at assessment), rather than inventing partial-module overlays or build-time generated sources. Only Windows frontend startup and production/dev app startup propagation receive behavioral edits, plus focused regression helpers/tests and provenance. Retain upstream Linux/macOS bytes. This is a patch to the existing framework, not a framework replacement.
- Add an error-returning binding Embed API while retaining the existing bool API compatibility. First failure wins; fail synchronous requests, failed/null asynchronous results and interrupted initialization before dereferencing outputs or publishing success. Unwind callbacks before normal owned-window teardown; reuse existing owner-thread cleanup.
- A fresh valid process is the recovery contract; repeated complete Wails.Run calls within one process are not introduced.

## Complexity Tracking

No constitution exception. The additional maintained module is limited to the demonstrated framework propagation boundary; its unchanged baseline is mechanically verified and the behavioral allowlist remains small. No new service, runtime, dependency version or policy framework.
