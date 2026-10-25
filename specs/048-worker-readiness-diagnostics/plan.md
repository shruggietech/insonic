# Implementation Plan: Managed worker readiness diagnostics

**Branch**: `codex/048-worker-readiness-diagnostics` | **Date**: 2026-10-25 | **Spec**: [spec.md](spec.md)
**Tracking**: #174, bounded increment of #147; milestone 32.

## Summary

Add an optional validated probe observation to existing inspection/receipt contracts, correct cancellation/deadline attribution through receipt/reuse/readiness, and expose the same fixed diagnostic to CLI, GUI and failed durable setup.

## Technical Context

Go 1.27.1 shared runtime; Python 3.12 managed probe; React/TypeScript wrapper; JSON Schema 2020-12. Existing workers/process/runtime and catalog work checkpoints are reused. No dependency or release version change. Windows/Linux x86-64 remain supported. Production probe/readiness defaults stay 60 seconds and the existing IPC wrapper stays 65 seconds. CI qualification remains 1800-second target / 3600-second hard after readiness.

## Constitution Check

Pre-research and post-design gates pass: owner-approved bounded scope; CLI-first shared behavior; immutable environment and source identity; no new routing/credentials or item review; hidden noninteractive children; required integrity, privacy, cancellation and functional checks. No constitution exception is required. #147's other audit dimensions remain open.

## Project Structure

- `internal/workers/`: optional probe observation, private checkpoint, error mapping, frozen receipt validation/reuse.
- `internal/app/workers.go`, `worker_readiness.go`: failed setup checkpoint, partial readiness and cancellation.
- `internal/runtime/`, `desktop/bridge.go`: ephemeral disconnect cancellation without changing durable work ownership.
- `internal/contracts/operator.go`, `schemas/v1.0.1/`: fixed reasons and optional bounded inspection schema.
- `cmd/insonic/human_summary.go`, `desktop/frontend/src/workers.tsx`: concise role/stage/reason/timing summaries.
- `scripts/journey_cpu.py`, maintained real journey: same-mode probe observations and actual managed CPU qualification.
- Focused Go/Python/Node/frontend/schema regressions; official contract prose under `docs/v1.0.1/`; compact internal verification here.

## Research and Design Decisions (2026-10-25)

1. D001: Use a small fixed-name atomic progress checkpoint inside the already private owned probe home. Validate bounded JSON and fixed labels only after the owned child stops. Prefer this to raw stderr parsing or generic streaming-process architecture.
2. D002: Keep the existing 60-second production envelope. Report expiry and unattempted roles truthfully; preserve already observed roles. Do not launch more work after expiry. Parallel engine imports add resource pressure without addressing expensive hashing and are not selected.
3. D003: Retain failed inspection in the existing setup work checkpoint before returning its fixed safe error. Register fixed reason strings so SafeError preserves them. A single owned-process Started fact in internal/process/capture.go distinguishes launch failure from exit failure without inferring launch from an exit-code sentinel. Cancellation must not become checksum mismatch, environment change or a preparation cache miss.
4. D004: Optional probe fields validate strictly when present. Existing receipts remain unchanged and readable; every original byte, compatibility and inventory check remains required.
5. D005: Client disconnect cancels only ephemeral worker inspection/readiness requests. Durable setup still requires explicit work cancellation.
6. D006: Present a compact shared diagnostic in human CLI and desktop, with native/child clock origins explicit. Do not expose exception strings, raw child output or arbitrary paths.
7. D007: Label first managed snapshot verification and repeated inspect-mode measurements separately. Compare identical modes on the same immutable environment; do not claim flushed OS caches or retrospective S046 causality.
8. D008: Use the maintained outside-CI real CPU journey on a clean source-bound complete CLI package. Preserve its CI guard, isolated profile and cleanup. Retain compact audited identities/outcomes on GitHub, then remove owned scratch.

## Delivery and Verification

Write failing controlled regressions before implementation. Run focused workers/app/runtime/CLI/bridge tests, Python probe fixtures, schemas, frontend checks/tests/build, root check/test and affected site checks. Inspect real child timeout/cancellation and malformed progress. Verify legacy receipts and genuine changed bytes. Commit clean implementation, publish official PR automatically, qualify current source and run the isolated real CPU campaign. Address at most two Codex rounds, all individual comments and final exact-head CI. Stop before merge.

## Convergence decision (2026-10-25)

D009: Name the fixed validated child history `checkpoints`. The generic catalog privacy contract rejects raw `observations` at any depth; preserve that guard without exceptions. The real managed campaign exposed the collision, and an actual catalog setup checkpoint/publication regression reproduces it before repair.

D010 (2026-10-25): The hosted real Arcade functional fixture failed after 25 seconds while using the production 20-second HTTP client. Apply the maintained request allowance only to that fixture client; retain production defaults, actual ordered-reference assertions and earlier caller cancellation. A virtual HTTP regression contrasts production expiry, delayed fixture success and earlier caller expiry.

D011 (2026-10-25): Qualification must await the existing correlated owned-runtime startup report before observation commands. Validate its fixed fields and workspace/startup IDs, keep the original readiness/overall budget, retain bounded startup receipts, and reject malformed, failed or late reports. This addresses a reproduced missing join; it does not establish the underlying failed-launch OS error or historical cause.
