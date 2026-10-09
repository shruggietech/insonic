# Implementation Plan: S012 Model references and acquire-on-demand

**Branch**: `codex/012-model-references-acquisition` | **Date**: 2026-10-09 | **Spec**: [spec.md](spec.md)

## Summary

Complete #34 using the existing verified model registry, typed portable catalog and durable scheduler. Add revisioned aliases/sources, compatibility inspection, immutable resolution and shared acquisition dependencies for processing and import. Preserve speaker-version lineage and hosted references without implementing matching/training.

## Technical Context

Go shared runtime and CLI; Wails/React/TypeScript desktop; Python native qualification and Node schema/site checks. SQLite/PostgreSQL catalog, filesystem/S3 artifacts and LadybugDB/ArcadeDB graph projections remain equal supported adapters. No additional inference dependency or model engine is introduced. CI stays bounded below ten minutes using existing affected-component caching and parallel jobs. All source responses, aliases and bundles have explicit bounds; waiting parents consume zero of four worker slots.

## Constitution Check

Pre-research and post-design PASS: owner scope remains #34; no hypothetical usage policy or per-item reviews. CLI-first shared contracts include desktop UX and documented advanced controls. Exact source/model lineage, unknown outcomes, no fallback, opaque credentials and all backends retained. Normal schema/snapshot compatibility is separate from nonexistent pre-v1 bulk audio migration. Public docs remain self-contained; software/docs/schema version remains 0.0.0. Windows tools use verified hidden launchers. Required checks exclude acoustic model loading/download/inference. Explicit push/PR authorization supersedes the skill's default pre-push halt; merge/release remains owner-controlled.

## Project Structure

Internal slice: spec.md, checklists/, research.md, data-model.md, contracts/, quickstart.md, tasks.md and verification.md.

Source: `internal/catalog/` for alias/source domains and frozen schema compatibility; `internal/models/` for reference/source/compatibility; `internal/app/` for operations, elections and dependencies; `internal/library/` and `internal/pipeline/` for input parity; `cmd/insonic/` and `desktop/frontend/` for UX; `schemas/v0.0.0/`, `docs/v0.0.0/`, `scripts/qualify.py` and existing tests for contracts/qualification.

## Decisions (2026-10-09)

- D1 Use lowercase 1-64 ASCII aliases beginning with a letter; unprefixed UUID and `base:UUID` select base installations, `speaker:UUID` selects existing trained versions. Explicit typed hosted targets preserve endpoint/handle/provider metadata. Alternatives (arbitrary names, implicit first name/version) are ambiguous.
- D2 Add revisioned alias/source domains with tombstones and operation replay/CAS; preserve schema-7 opening and snapshot proofs before schema-8 additions. Alternative duplicate model tables would destroy existing lineage.
- D3 Optional compatibility declaration records adapter/version, architecture/runtime and audio requirements. Legacy manifests retain bytes/digests and infer only demonstrated role contracts. Unknown/custom models remain registerable/acquirable but unsupported for execution.
- D4 Configured bounded manifest catalogs provide discovery and selector-to-pinned-manifest resolution. Source URLs and opaque credentials follow existing transport rules. Direct HF integration is not claimed because external CDN redirects and auxiliary Git SHA-1 metadata need a distinct provider contract. Full bundle SHA-256/size pins remain mandatory.
- D5 Resolve aliases/sources once during submission. Durable payload keeps exact resolved models, digests and acquisition work IDs. Saving/inspection remains side-effect free; discovery is distinct from registered/available state.
- D6 Deduplicate explicit/automatic acquisition by workspace/model/manifest-derived stable work identity. Gate parents before claim; never block all four workers while waiting for downloads. Dependency failure/cancellation becomes a diagnosed parent failure; parent retry restarts the same frozen acquisition. Cancelling a parent does not cancel shared acquisition.
- D7 Verify available artifacts and repair corruption/missing bytes through exact acquisition. Preserve leases and trained/profile durable data. Reconcile identical available publication where necessary.
- D8 Freeze import model identity/digest separately; update library proofs and apply digest to import diarization rather than only fixing recordings.process. No changing identity on partial batch retry.
- D9 Keep trained identity/lineage and hosted handle inspection faithful; adapters may diagnose unavailable/unsupported execution. Full #35/#15 are not completion claims.
- D10 Custom checklist markers remain reviewer-owned and untouched. Autopilot requirements review found CHK001-010 covered by spec/design; its routine proceed decision follows the owner's end-to-end authorization and the orchestration decision policy. No checklist marker is treated as implementation evidence.

## Execution and Verification

First implement typed catalog/reference contracts and meaningful failure tests, then scheduler/election/import integration, then CLI/desktop/schema/docs and qualification. Sequential tasks touching shared files stay sequential; independent tasks marked [P] may execute in parallel as directed by speckit-implement. Required commands: npm run check, npm test, Go tests/vet and affected race checks, frontend check/test/build, site build, Python checks and bounded CLI/desktop/native package journeys. Alternative-backend CI exercises PostgreSQL/S3/ArcadeDB. No real weights or inference even through cached models.

Official PR closes #34 only after all acceptance obligations are implemented, documents remaining #35/#15/#14 work, and is attached to this chat. Inspect every comment/thread/reaction and exact-head CI, answer/resolve findings and use at most one second request. Stop for owner final review, never merge.

## Complexity Tracking

No constitution exceptions. A configured catalog protocol is a proportional provider adapter, not a new model subsystem. Exact trained/hosted references preserve existing entities rather than inventing training output.
