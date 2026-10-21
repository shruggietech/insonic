# Implementation Plan: CLI readiness and failure reporting

**Branch**: `codex/043-cli-readiness-reporting` | **Date**: 2026-10-21 | **Spec**: [spec.md](spec.md)

## Summary

Repair six related truth/reporting defects in existing shared services, then exercise their human/JSON/durable behavior through the current packaged CLI qualification. No new execution framework or release is introduced.

## Technical Context

- Language/version: pinned Go 1.27.1; existing Python qualification and TypeScript GUI consumers.
- Dependencies/storage: current contracts, artifact, workers, models, secrets, pipeline, runtime and voicemodels packages; filesystem/S3 and existing catalog adapters remain supported.
- Testing: focused Go tables/security tests, existing isolated Python CLI regressions, required npm checks/tests and affected frontend/site checks; ordinary complete hosted qualification.
- Platforms: Windows/Linux x86-64. Local console children use the verified hidden process owner and pinned toolchain; GUI product tests run in hosted qualification.
- Performance: preserve operation bounds, 660s qualification target/900s hard limit, verified readiness origin and separate preparation/qualification/total accounting.
- Scope: #126 and #135-139; only relevant fixture subsets from #114. #147 and historical #154 remain open.

## Constitution Check

Pre-research and post-design: pass. Shared CLI/runtime authority, protected credentials, explicit observed/unknown state, no routing/eligibility changes, no owner input/GUI gate, no weakened integrity checks, no dependency/version upgrade, no automatic merge/release. Product/docs/schema stay 1.0.1. Public docs describe contracts; internal slice artifacts stay under specs.

## Project Structure

- Shared failure vocabulary: internal/contracts/contracts.go, operator.go and a focused asset_error.go; schemas/v1.0.1/runtime-response.schema.json.
- Integrity: internal/artifact/{contract,filesystem,s3,service}.go and internal/voicemodels/service.go.
- Local assets/budgets: internal/workers/{setup,types}.go, internal/models/service.go, internal/app/workers.go.
- Readiness: internal/app/worker_readiness.go and existing pinned trainer checks in internal/voicemodels/adapter.go.
- Wait: cmd/insonic/work_wait.go and existing human rendering.
- Consumers/tests: relevant package tests, desktop/frontend/src/workers.tsx, worker plan types, scripts/cli_contract_qualification.py and scripts/documented_workflows.py with existing Python tests.
- Documentation: docs/v1.0.1 processing/error/CLI contracts and CHANGELOG.md; concise tracked Spec Kit artifacts only.

## Design and execution

1. Add bounded Error.asset metadata with fixed role labels and validated ASCII slash-relative logical names (maximum 128 bytes). Fixed allowlisted reasons encode verification stage and correction. SafeError reconstructs these fields; raw errors/paths/tool output never cross the boundary.
2. Correct known physical object integrity categories on filesystem/S3 while preserving missing catalog IDs, unavailable access, real revision fences and pending-upload observation/reconciliation. Add PublishVerified sharing existing staging/upload code, checking the immutable scratch digest/size before BeginPublication. Trainer output uses it instead of publish-then-retire for a false digest.
3. Reuse local asset verification in preview and execution. Report generated missing/unreadable/not-regular/size/hash/tool causes with logical names. Offline budgets expose required/verified/missing local totals and zero elected download bytes; retain a separately labeled nominal profile total.
4. Check local credential resolvability only for actually required downloads, deduplicate references, clear resolved byte slices immediately and report upstream access not_checked. Preserve installed/local reuse, frozen destination and original credential selection.
5. Reuse pinned executable/support hashing without running a trainer. Keep hosted/runtime protocol status unobserved and compose pipeline usability from elected operations, including training-only definitions.
6. Keep one caller wait context across Ensure and polling; normalize caller-driven expiry/cancellation at the wait boundary, preserving independently observed startup failures and actual last work state. Timeout advice makes no unobserved running/completed claim.
7. Extend existing deterministic/package CLI paths for all six issues, including a tiny pinned local trainer fixture and existing loopback training fixture. Assert error codes/reasons, human/JSON consistency, durable causes, no invalid publication, no secret/path leakage and owned cleanup.

## Dependencies and boundaries

The error contract precedes service changes and schema/renderer assertions. Integrity, local-asset/readiness and wait stories are independently testable after that foundation; shared app/qualification files are edited sequentially. No broad audit-corpus replacement, historical incident campaign, new worker process model or review bot framework is added.

## Decisions (2026-10-21)

See research.md for evidence and alternatives. Local expectations are additive where practical; corrected error-category and timeout-message meanings are documented as behavior corrections. All checks must bind the final PR source. Initial automatic reviews are round one; at most one explicit second-round request is allowed. Stop at an unmerged green PR for owner review.
