# Implementation Plan: Real CPU processing and speaker matching

**Branch**: `codex/046-real-cpu-processing` | **Date**: 2026-10-23 | **Spec**: [spec.md](spec.md)

## Summary

Extend the maintained qualification path for #147 with exact-package real CPU execution, independent reference comparisons and compact reproducible evidence. Publication is owner-authorized; merging is not.

## Technical Context

**Language/Version**: Python 3.12 maintainer tools; Go 1.27.1 product; Node 22+ verification.

**Primary Dependencies**: Existing pinned faster-whisper/Community-1 CPU engines, managed worker profiles and complete Windows package companions.

**Storage**: Disposable filesystem/SQLite/LadybugDB libraries and portable archives through existing runtime operations.

**Testing**: Root npm check/test, Python maintainer tests, affected Go/site checks, separate elected real-engine run, exact-head hosted native/backend/package CI.

**Target Platform**: Windows x64 CPU real acceptance; existing Linux/Windows required matrix remains unchanged.

**Project Type**: Maintainer qualification for shared CLI/runtime operations.

**Performance Goals**: Bounded fixture and operation budgets; ordinary required qualification targets 660 seconds after readiness with a 900-second hard limit.

**Constraints**: No owner GUI, owner library mutation, new release or third review round; hidden noninteractive children, exact provenance and complete cleanup.

**Scale/Scope**: At least two enrollment identities, disjoint held-out speech and one unseen speaker plus a two-speaker sequence.

**Resolved research**: Official LibriSpeech source and hashes; existing connected-journey methods; builtin managed/model APIs; local exact model snapshots require product hash verification; clean current package must be rebuilt. See research.md.

## Constitution Check

Pre-research gate passes: owner-approved observable scope; shared CLI authority; immutable source/reference provenance; protected credentials and configured routes; separately elected inference and unchanged required CI; compact internal evidence, no public slice links; no merge/release/owner-desktop execution. No new usage policy or per-item approval is introduced.

## Project Structure

`scripts/qualify-journey.py` remains the single journey entry point. Add `scripts/acoustic_reference.py` for corpus/provenance/scoring and `scripts/journey_cpu.py` for the reference-backed real stages using existing Journey methods. Extend `scripts/qualify-processing.py` only for shared bounded ownership/atomic evidence. Add focused Python tests, `tests/fixtures/acoustic/` originals/manifest/attribution and a binding in the maintained fixture index. Update maintainer documentation and changelog. No runtime/schema/GUI API expansion is planned; fix demonstrated integration defects only.

## Implementation decisions

- Add optional `--reference-corpus` for the package-only real journey. Legacy modes retain their explicit arguments; managed corpus mode obtains interpreters through the product. `--managed-offline-assets` optionally elects a complete verified dependency bundle. Both existing model snapshot flags select exact local inputs, never mutable model aliases.
- Corpus roles, model IDs and matching settings are frozen before model invocation. A generated two-speaker WAV lives only in owned scratch with integer sample offsets. Enrollment maps every predicted local voice in each independently single-reader clip to that reference reader; held-out files are never enrollment inputs.
- A compact versioned receipt has separate functional stage acceptance and acoustic measurements. Completeness requires all expected corpus cases, stage identities, restore comparisons and observed cleanup; every failure preserves its first stage/code. Partial/synthetic/stale receipts cannot pass.
- Isolated real acceptance uses a clean complete Windows CLI archive under a Unicode path with spaces. Rebuild after implementation commit; verify canonical source/package identities, retain compact public evidence, delete temporary copies. Keep preparation versus execution versus total measurements distinct from hosted CI qualification.
- Required CI remains inference-free. Test malformed/missing/changed inputs, overlap, normalization/edit counts, speaker decision accounting, receipt completeness, mode selection, first failure and descendant cleanup. Run existing complete checks and exact-head hosted CI; allow at most two Codex rounds.
- Local package execution demonstrated that a 30-second `work wait` can return a recoverable pending result. Repair the package smoke join to continue only that exact work identity under the original 180-second operation bound, retaining 30-second requests and every hosted CI deadline. Add deterministic deadline/identity/error regressions; no product API or deadline expansion is involved.
- A focused real cold-bootstrap reproduction returned the correct refusal in 0.875 seconds after owned launch; its launcher took 1.968 seconds. Separate those measurements in scripts/cli_contract_qualification.py, retain the two-second process classification limit and report total invocation time. Hosted CI still counts the entire job. Regress source drift and final cleanup-stage attribution before the elected CPU campaign.
- The first managed real processing case failed invalid_timing after successful setup/model/readiness stages. A separate explicitly noncanonical CPU diagnostic observed a Community-1 turn at 13.767–13.970s against 13.690s decoded audio: zero source overlap. Discard and diagnose empty intersections before applying the existing 250ms overlapping-overhang guard. Preserve valid activity, strict generic turns, frozen references/settings and raw count bounds. Retain numeric diagnostics/boundary policy and assert exact provenance/diagnostic recovery.
- The built-in author checklist covers domain completeness and is complete. No additional reviewer-owned checklist is generated; the existing autopilot authorizes routine phase progression.

## Constitution Check after design

Pass. Corpus-specific inputs remain test fixtures/internal planning, not product eligibility or public application defaults. Models are genuine pinned CPU engines; enrollment is not base-weight retraining. Statistical failures remain explicit measurements. Existing source integrity, backend support, inference-election and CI timing contracts are retained. Public docs explain the maintainer contract without numbered planning links. No merge, release or owner GUI is performed.
