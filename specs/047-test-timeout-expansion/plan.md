# Implementation Plan: Comprehensive test timeout expansion

**Branch**: `codex/047-test-timeout-expansion` | **Date**: 2026-10-24 | **Spec**: [spec.md](spec.md)

**Input**: S047, issue #172, milestone 31.

## Summary

Expand successful-execution allowances at every active testing layer, with shared stage policy, validated aggregate policy, coherent parent budgets and fixed monotonic help-progress observations. Preserve normal product deadlines and short injected failure fixtures. Publish automatically, verify exact-source full CI and cold/warm preparation, resolve at most two Codex rounds and stop before merge.

## Technical Context

**Language/Version**: Existing Go 1.27.1, Python 3.11+, Node 22.12+, TypeScript.

**Primary Dependencies**: Existing Wails/WebView2, React, Playwright and Python standard library; no dependency update.

**Storage**: Committed policy and compact deadline inventory; GitHub Issues, milestone and Project remain delivery authority.

**Testing**: Root check/test, affected Python/Go/frontend/site suites and hosted full native/package/backend acceptance. All local process launches stay hidden and noninteractive; native GUI/input only on disposable hosted runners.

**Performance Goals**: 1,800-second advisory / 3,600-second hard qualification after readiness, 60-second finalization reserve; >=65-minute job watchdogs and >=60-minute preparation jobs.

**Constraints**: Named seconds units, original absolute parent/phase deadline, no clock restarts, prompt success/cancellation, bounded diagnostics, natural-release truth, exact artifact identity. Existing ordinary production clocks remain unchanged.

**Scale/Scope**: All maintained test entry points and wrappers, including embedded PowerShell/native waits; no general-purpose scheduling framework.

## Constitution Check

The owner expressly authorizes expanded time allowances in #172. A separate constitution amendment to 5.0.0 redefines principle V before analysis; the prior 660/900 rule is historical. Required functional, security, source/package and elected checks are preserved. Exact product version remains 1.0.1. No owner desktop/input, installation, release or merge is authorized.

Pre-research and post-design gates pass. No public runtime API or storage change is needed. Internal policy/observation entities are described in data-model.md; no separate public contract artifact is warranted.

## Project Structure

- `internal/qualificationtiming/policy.json`: canonical test-stage seconds; Go embeds the same bytes.
- `scripts/qualification_timing.py` and `scripts/qualification-timing.mjs`: validated Python/JS access.
- `scripts/ci-timing-policy.json`: existing aggregate target/limit/reserve contract, updated without confusing it with job cancellation.
- `cmd/insonic-desktop/`, `desktop/`, frontend qualification: fixture-only limits and bounded monotonic help stages.
- `scripts/`, `.github/workflows/`, site verification: complete caller/wrapper/job integration.
- `tests/`, frontend and Go tests: delayed success, expiry, cancellation, policy/matrix relations and existing assertions.
- `specs/047-test-timeout-expansion/`: concise decisions, inventory, task and acceptance references.

## Phase 0 decisions

See [research.md](research.md). Spec Kit research agents inspected native/frontend, Python helpers, and workflows/browser tooling. A 25-second native help watchdog currently precedes Python's 35 seconds. Twelve sequential Windows Exit cases and installer actors need larger enclosing budgets. Cache identity must include any newly consumed policy input.

## Phase 1 design

Use one small flat numeric seconds policy with explicit stage names, positive finite integer validation and cross-layer relation tests. Embed it in Go, import it in TypeScript, and load it in Python/JS; no ambient user runtime configuration is added. Test overrides remain explicit. Increase child/matrix defaults above #172's minimum when useful, and cap sequential stage waits by an explicitly reported earlier enclosing deadline rather than claiming every per-stage maximum is additive.

Use monotonic native elapsed time and separate frontend monotonic time for fixed allowlisted stages. Received native observations do not prove a frontend RPC response was consumed. Preserve bounded stdout/stderr and nonzero timeout outcome. Inventory entries group actual timing owners, caller origins, parent bounds and concrete exceptions; generated dumps of investigative thoughts are unnecessary.

## Implementation sequence

1. Foundation: validated policies/loaders, injected regression tests and checked inventory.
2. Parallel file-owned integration: native/frontend Go/TS; Python packaging/helpers; workflows/Node/browser tools.
3. Parent integration: governance/docs/changelog, policy consistency/inventory, focused checks and required suites.
4. Commit/push/open official PR; wait for automatic Codex/security reviews and CI, repair failures, answer/resolve each actionable thread.
5. After final implementation source is stable, run isolated cold full CI and rerun that entire run for same-source warm comparison; independently audit terminal attempts and artifact digests.
6. Converge, update compact tracking/evidence, verify exact final head, checks and review cap, then stop before merge.

## Verification and tradeoffs

Tests use fake clocks/delayed owned children for former-bound success and new expiry; retain intentionally short deadline/cancellation fixtures. Normal successful execution returns promptly, so expanded upper bounds do not mandate slower passing tests. Genuine stalls may take longer and consume more hosted resources. Any surviving startup failure remains #154 with observed stage evidence; one passing run is not a causal closure.

## Complexity Tracking

No unresolved constitutional violation. The 5.0.0 amendment is explicit owner-directed governance work; public test policy documentation is synchronized during implementation.
