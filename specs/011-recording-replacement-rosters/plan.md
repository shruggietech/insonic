# Implementation Plan: S011 Recording replacement and declared speaker rosters

**Branch**: `codex/011-recording-replacement-rosters` | **Date**: 2026-10-09 | **Spec**: [spec](spec.md)

## Summary

Complete #30/#33 using an elected extension of the existing canonical admission transaction. Add #35's declared roster foundation, import and replacement lifecycle, leaving acoustic matching explicit unfinished work. Start from merged bfb2ad1. No populated pre-v1 audio migration is required.

## Technical Context

Existing Go, SQLite/PostgreSQL typed catalog and shared current-recording receipts; filesystem/S3 verified artifacts, leased retirement, durable work; LadybugDB/ArcadeDB rebuild projection; Wails/React desktop and versioned JSON schemas. Reuse canonical FFmpeg/Cueson operations and rational clocks. No new storage subsystem, model runtime or dependency is needed.

Existing ingress 8 MiB/10,000 items and paged response bounds remain. Roster membership is bounded to 1,000 speakers, matching existing speaker-context bounds; selectors use existing identity-text maximum 512 UTF-8 bytes. No names/profiles/documents are copied into memberships. Media/transcript conversion bounds and hidden child-process guarantees remain unchanged.

## Constitution Check

Owner-directed replacement/context scope: pass. Shared CLI/runtime/desktop: pass. Canonical/source identity, captured facts, current evidence and leases: pass. Alternative adapters and credentials: pass. Model-free deterministic/native verification and ten-minute CI: pass. Amend superseded pre-v1 legacy-conversion directive separately before analyze, recording owner deployment correction and constitution 2.2.1 in changelog. No waiver of migration/schema integrity.

## Decisions

- D01 Extend CommitAdmission with typed optional admission elections while preserving existing callers. Only elected audio replacement may change source fields; generic CommitLibrary/UpdateLibrary remains immutable.
- D02 Always publish a new current Recording for replacement, including clear/audio-only, using truthful `untranscribed` state with null document. Source and document CAS are independent and both advance under one catalog transaction.
- D03 Keep preserves exact current document bytes after explicit applicability assertion and rational known-bound validation. Changed source removes prior acoustic/central mappings and reconciles current clips/datasets/preparation even when document bytes match. Native source-owned observations remain preserved.
- D04 Replace reuses independent explicit/embedded transcript acquisition; no valid selected transcript is an error for replace. Clear does not imply recognition. Independently elected processing follows accepted admission only.
- D05 Roster header anchors to LibraryEntry.ID before any processed document; explicit empty header differs from undeclared. Unique members reference centralized speaker IDs. Use current catalog receipt/proof/CAS patterns, not generic raw Commit admission.
- D06 References use UUIDs or exact canonical name/active alias. Equal names across identities fail. An explicit `name:` prefix can disambiguate UUID-shaped names; IDs resolve directly. Resolve once at enqueue/mutation, deduplicate IDs, recheck active elections transactionally. Existing inactive membership stays readable; remove/clear remains possible.
- D07 Roster mutations read/add/remove/replace/clear use expected independent roster revision. Idempotent set no-ops preserve roster revision, while receipts reconcile repeated operation identity. Omitted imports leave undeclared; explicit [] declares empty. Retain/clear on audio replacement is required only when a roster header exists; supplied replacement membership conflicts with retain/clear and is rejected.
- D08 Freeze target/library/current-document/roster revisions and resolved speakers in import input before work. Accepted markers/receipts bind roster intent and current authority without source locators or duplicated membership arrays in historical results.
- D09 Change media_entry_asset current source attachment and remove obsolete media_asset/metadata structural references only if no shared/current owner remains, then queue old publications for existing leased retirement. Preserve owner date choices, refresh source facts, and invalidate old playback/processing/extraction fences.
- D10 Add normal additive catalog roster schema migration and portable current proofs. Register typed domains and snapshot schema; reject missing parents, duplicate members, mismatched revisions and forged receipts on both catalog adapters.
- D11 Graph rebuild includes distinct declared-speaker edges and undeclared/empty roster diagnostics, using generic entity/edge projection. Membership never enters cue assignment, corpus or acoustic evidence selection.
- D12 Keep `insonic media import`/`transcript import`; add targeted media flags and repeatable known-speaker references, JSON/CSV equivalents and `recordings roster` family. Desktop uses shared operations with accessible explicit controls. No added per-item confirmation.
- D13 Full #35 matching, #34 models, #15 training and #14 release remain outside this slice. #30/#33 closure requires all their matrix/retention/replacement criteria plus the roster import foundation, not merely new tables.
- D14 Initial official PR triggers review round 1. At most one second request is permitted; reply to every finding, verify resolved threads and exact final-head CI, and stop before merge.

## Project Structure

Catalog: internal/catalog/roster_records.go, roster.go, admission.go, library_state.go, recording.go, evidence.go, records.go, migration.go, validation/proofs. Library: types.go, admission.go, manifest.go, service/extract.go. Runtime: internal/contracts and internal/app/domain.go, work.go. CLI: cmd/insonic/domain.go/main.go. Graph: internal/explore/corpus.go. Desktop: frontend/src/screens.tsx/client.ts and rendered fixtures. Public: schemas/v0.0.0, docs/v0.0.0, changelog and constitution. Evidence stays in this directory.

## Phases and Verification

Requirements/checklist -> research/design -> tasks -> blocking analyze -> failing catalog/library tests -> foundation roster/current replacement -> interface integration -> docs/contracts -> affected Go/vet/race, root check/test, frontend/site and native/backend verification -> converge -> commit/push/official PR -> bounded reviews and final exact-head CI -> owner merge handoff.

Independent read-only research covered admission/cleanup, roster/portability and CLI/desktop/schema. Shared implementation is sequential. Independent checks can be batched, except package isolation and native tests must not overlap. No real acoustic models execute in required checks.
