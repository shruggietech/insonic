# Implementation Plan: Grounded speaker enrollment and unattended application

**Branch**: `codex/073-grounded-speaker-enrollment` | **Date**: 2026-11-02 | **Spec**: [spec.md](spec.md)

**Input**: S073 #230-#232, milestone 49, existing epic #211. Automatic push/PR, two review rounds maximum, owner merge; no new tracking issue.

## Summary

Extend the existing mapping, corpus and enrollment authority with independently grounded context and add a durable configured speaker workflow. Keep source text immutable, corrections authoritative and immutable model outputs retrievable independently of current profile usability.

## Technical Context

**Language/Version**: Source-pinned Go 1.27.1; existing Node >=22.12 and Python maintainer tools.
**Primary Dependencies**: Existing catalog transactions/receipts, speakerdiscovery resolution, voicemodels preparation/enrollment, assistance provider, pipeline and scheduler. No new external dependency.
**Storage**: SQLite/PostgreSQL existing mapping provenance and work/receipt state; filesystem/S3 outputs; LadybugDB/ArcadeDB recoverable projections.
**Testing**: Focused protocol/catalog/voice-model/runtime/CLI tests, failure and snapshot tampering cases; Go test/vet, npm run check, npm test, affected docs/site/frontend, package/backend/native qualification in CI.
**Target Platform**: Windows/Linux x86-64, shared CLI/desktop runtime.
**Project Type**: Existing Go CLI and desktop application.
**Performance Goals**: Existing discovery bounds (500 cues, 512 observations, 1000 voices/candidates, 1 MiB aggregate), 100000 dataset references, four active scheduler workers. Parent workflows must yield while children run, never occupy all slots waiting.
**Constraints**: No per-item approval, no silent provider routing, encrypted credential references, source/mapping/profile revision fences, hidden noninteractive child processes. Qualification target 1800 seconds, hard limit 3600 seconds after readiness, separate preparation and retained receipts.
**Scale/Scope**: Grounded enrollment, correctable application/retraction and a coherent saved unattended workflow; real corpus quality/release readiness remains #233.

## Constitution Check

Pre-research: principles I-V pass. Preserve requested scope, shared CLI/runtime, source lineage, supported backends and autonomous configured execution. Three read-only planning agents examine catalog authority, enrollment and workflow orchestration under the installed planning command.

Post-design: no exception. Technical independence checks distinguish evidence classes; they do not add approval gates. Existing mapping authority is extended without a parallel assignment store. Cancellation and retries preserve accepted stages. Historical outputs remain immutable; current usability is checked separately. No publication beyond the authorized PR, and no merge.

## Project Structure

Planning: specs/073-grounded-speaker-enrollment/{spec,plan,research,data-model,quickstart,tasks,verification}.md and contracts/runtime.md. Runtime: internal/catalog/speaker_application.go, existing selection/model/discovery/work authority; internal/speakerdiscovery/; internal/voicemodels/; internal/app/speaker_workflow.go; internal/pipeline/definition.go; internal/contracts/; cmd/insonic/. Public contracts: schemas/v1.2.0/, docs/v1.2.0/, CHANGELOG.md. Extend existing maintained checks in tests/ and scripts/ where installed workflow coverage requires it.

## Decisions and execution order

1. Freeze server-derived contextual lineage in the existing mapping provenance; discovery snapshots may be retired after application and are not required as ongoing enrollment authority.
2. Distinguish manual, acoustic automatic and independently contextual mappings. Match-derived or circular assertions never acquire contextual origin. Preserve manual corrections and independently grounded mappings during matching.
3. Apply/retract all decisions for one recording atomically with the discovery and source revisions fenced; use receipts for unknown commit reconciliation. A rerun reconciles obsolete contextual decisions while retaining manual corrections.
4. Validate contextual currentness against exact recording/source/document and supporting speaker identity. Invalidation must occur before output/work reconciliation so dependent state changes in the same transaction.
5. Extend every corpus-selection and portable provenance boundary consistently, retaining historical manual-only digest compatibility unchanged.
6. Separate a durable speakers.run coordinator from terminal speakers.discover work. Reuse child discovery/training, deterministic child operation IDs and persisted stage results; yield scheduler capacity between stages. Fence child effects against cancelled/stale parent authority and propagate cancellation/retry appropriately.
7. Save explicit speaker_workflow configuration alongside existing speaker_training; freeze exact pipeline revision, provider configuration, model/device and tools before execution. No alias/config retargeting on retry.
8. Expose apply, retract and run with full versioned JSON and concise evidence-class human output. Keep desktop shared dispatch aligned and document dedicated-control lag.

## Workflow notes

No extension hooks are installed. setup_plan initialized the required skeleton before the domain checklist (the checklist helper requires plan.md). Requirements checklist passes 16/16; custom authority checklist retains 12 unchecked reviewer-owned markers under explicit autopilot authorization. Branch, feature directory, selector and issue scope all identify S073.
