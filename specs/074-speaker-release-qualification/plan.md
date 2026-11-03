# Implementation Plan: Speaker quality and release qualification

**Branch**: `codex/074-speaker-release-qualification` | **Date**: 2026-11-03 | **Spec**: [spec.md](spec.md)

## Summary

Complete existing #233 and parent #211 with separate contextual and acoustic measurements, installed lifecycle/backend acceptance, and an unpublished 1.3.0 candidate. Use real local model execution on CPU/NVIDIA and owned qualification workspaces. Preserve deterministic required CI and historical CPU acceptance contracts.

## Technical Context

**Language/Version**: Go 1.27.1, Python 3.11+, Node 22+, existing React/Next.js frontend/site.
**Primary Dependencies**: Existing CLI/runtime, Python process ownership, pinned media/Cueson companions, offline pyannote community-1, local Ollama qwen2.5:14b, frozen acoustic references and release tooling.
**Storage**: Owned filesystem/SQLite/LadybugDB evaluation workspace; existing PostgreSQL/S3/ArcadeDB installed CI campaigns. No new product store.
**Testing**: Failing-first corpus/metric/election/receipt tests; root, full Go/vet, Python, frontend/site checks; real contextual/CPU/CUDA campaigns; hosted native/package/installer/backend checks.
**Target Platform**: Windows/Linux x86-64 product packages; local Windows headless model campaign and isolated hosted UI capture.
**Project Type**: CLI-first product qualification and release preparation.
**Performance Goals**: 1800-second qualification target, 3600-second hard limit, 60-second provisional finalization reserve; separate preparation/qualification/combined accounting.
**Constraints**: No new issue, owner merge, no tag/release/docs promotion, two review rounds maximum. No local GUI, interactive prompts, credential leakage or implicit model routes. Ordinary product deadlines remain intact.
**Scale/Scope**: Eight contextual scenarios, five frozen licensed acoustic clips (two enrollment, two held-out, one unseen), CPU/CUDA matching rows, full installed lifecycle/backend checks, four product variants and ten release files.

## Constitution Check

Pre-research PASS across all five principles. Scope is owner approved; existing contracts, uncertainty, evidence independence, credentials and supported backends persist. Qualification explicitly distinguishes protocol and model quality. Publication is bounded by current authorization. Post-design PASS: no constitution exception or unapproved product restriction.

## Phase 0 Research

Three installed-plan research agents examined acoustic readiness, evaluation contracts and release integration. [research.md](research.md) records decisions and alternatives. Existing cached models and CUDA readiness are available; this does not establish inference. Ollama needs an owned hidden loopback server. No unresolved design choice requires owner input.

## Phase 1 Design

1. Freeze eight synthetic contextual scenarios with supplied voice truth and a non-speech PCM carrier, explicitly separate from the licensed acoustic source. Expected outcomes never enter discovery requests.
2. Implement bounded strict reference validation and metrics with numerator/denominator/null-when-undefined, complete per-case accounting and honest terminal errors. Model accuracy is measured, not constrained to an invented universal threshold.
3. Reuse installed CLI operations and owned runtime/process/deadline helpers for discovery/resolution. Record configuration and immutable local model identities before/after execution. Deterministic provider fixtures remain model-free and separately labeled.
4. Reuse frozen acoustic corpus validation and the connected Journey infrastructure for real product profile enrollment and held-out matching with supplied known activity. Explicitly elect cpu then cuda. Preserve journey_cpu historical contracts; its helpers omit device and cannot be reused unchanged as CPU proof.
5. Verify actual model/device attempts and independent source splits. Report supplied-activity enrollment/matching quality, not ASR or diarization accuracy. Keep contextual and acoustic observations separate in the acceptance packet.
6. Retain existing installed configured speaker workflow, anti-circular, correction/retraction, failure/retry/restart and alternate-backend CI journeys. Add focused regressions only for demonstrated gaps.
7. Prepare 1.3.0 using maintained release.py. Copy current docs/schema forward, restore complete published v1.2.0 trees/navigation, document catalog 11-to-12 upgrade/downgrade behavior, and preserve independent discovery protocol version 1.
8. Bootstrap clean new-version hosted UI captures, retain/audit the run before any rerun, import the original 25 PNGs/four receipts, restore matching embeds and verify provenance. Never relabel old images.
9. Complete local parity and Spec Kit convergence. Open the official PR, attach #233/#211 and Project 7, handle review within two rounds, obtain terminal PR and exact-branch CI plus a validated complete candidate. Retain final evidence in the PR to avoid self-referential source claims.
10. Reconcile existing issue/epic acceptance with evidence and hand off for owner merge. Candidate package inference metadata remains not-executed because required package checks run no models; separate elected receipts establish actual inference.

## Project Structure

- specs/074-speaker-release-qualification/: specification, design, tasks, highlights, verification and compact receipts.
- scripts/speaker_reference.py; tests/fixtures/speakers/manifest.json; tests/speaker_reference_test.py: contextual truth and independent metrics.
- scripts/qualify-speakers.py; scripts/speaker_quality.py; tests/speaker_quality_test.py: maintained model-free/real evaluation, CLI integration and device evidence.
- scripts/acoustic_reference.py; scripts/qualify-journey.py; scripts/journey_cpu.py; scripts/cli_speaker_workflow_qualification.py: reuse existing authority without weakening historical contracts.
- VERSION and owned version-bound manifests/runtime/tests; docs/versions.json; docs/v1.3.0; schemas/v1.3.0; docs/screenshots/v1.3.0; CHANGELOG.md: release preparation.
- Existing CI/release/package tooling supplies required hosted qualification and candidate aggregation.

## Complexity Tracking

No new product architecture, store, provider or policy. Reviewer-owned qualification checklist remains unchecked; explicit owner autopilot authorizes proceeding. No extension hooks exist. Checklist prerequisites required plan.md before checklist generation; its active template was resolved directly before ordinary plan setup.
