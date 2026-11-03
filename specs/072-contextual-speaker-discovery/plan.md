# Implementation Plan: Contextual speaker discovery and resolution

**Branch**: `codex/072-contextual-speaker-discovery` | **Date**: 2026-11-02 | **Spec**: [spec.md](spec.md)

**Input**: Owner-authorized S072 for #227-#229. Automatic push/PR and at most two review rounds; no new tracking issue; owner merge.

## Summary

Deliver current evidence hypotheses, structured elected-provider extraction and reproducible resolution through shared runtime and CLI operations. Preserve accepted mappings/profile authority; automatic application/enrollment remains #230-#232.

## Technical Context

**Language/Version**: Source-pinned Go 1.27.1, current Node >=22.12 and Python maintainer tooling; no new dependency intended.
**Primary Dependencies**: Existing assistance.HTTPAdapter.Generate, current catalog evidence/mapping/roster contracts, contracts runtime dispatch and current CLI.
**Storage**: SQLite/PostgreSQL typed current_speaker_discovery aggregate, catalog schema 12, receipt proofs and portable snapshot validation; existing managed artifacts remain unchanged.
**Testing**: Meaningful pure protocol/resolution tests; catalog isolation/migration/freshness/retry tests; shared app/CLI fixtures; Go test/vet; root/schema checks and tests; affected docs/site/frontend and hosted native/backend/package CI.
**Target Platform**: Windows/Linux x86-64 CLI and desktop shared runtime.
**Project Type**: Existing Go application with CLI and desktop bridge.
**Performance Goals**: One assistance TimeoutMS aggregate deadline; max_cues 100/500, max_calls 4/20, max_observations 128/512 defaults/caps, 16 references per observation, 1 MiB aggregate, existing prompt/context/transport limits. Coverage and omitted cues remain explicit.
**Constraints**: Hidden noninteractive child execution, no active-workstation GUI, elected provider/credential scope, current source authority, deterministic evidence independence, no per-item manual gate.
**Scale/Scope**: One coherent recording-scoped discovery/inspection/resolution journey for three issues; reuse authenticated accepted matching work and independently grounded current dataset lineage without invoking mapping mutation.

## Constitution Check

- I: Approved scope is preserved, including automatic contextual support; no eligibility restrictions or mandatory manual approvals.
- II: Core behavior lives in shared runtime contracts and has usable human/JSON CLI operations; desktop shared bridge stays aligned, with advanced control lag documented.
- III: Evidence binds current source/cue/voice authority; equal names never merge identities; stale/derived predictions cannot become independent proof; mappings/source remain unchanged by discovery.
- IV: Reuse configured provider routes and encrypted credential references. Preserve SQLite/PostgreSQL and supported storage/graph behavior without implicit fallback.
- V: TDD for security, isolation, integrity and recovery; maintain 1800 s target/3600 s hard qualification boundary, retained terminal receipts and owner-controlled merge.

Pre-research gate passes with no exception. Three read-only planning agents investigate catalog persistence, provider/runtime integration and acoustic/resolution provenance. Post-design gate passes all five principles with no exception: current evidence is fenced, both catalog engines share the contract, configured routing remains explicit, no manual gate is introduced and owner merge/publication boundaries persist.

## Project Structure

Planning and evidence: specs/072-contextual-speaker-discovery/. Existing implementation boundaries: internal/catalog/, internal/assistance/, internal/speakers/, internal/voicemodels/, internal/app/, internal/contracts/, cmd/insonic/, schemas/v1.2.0/, docs/v1.2.0/, scripts/ and tests/. New files: internal/speakerdiscovery/{types,parse,resolve}.go; internal/catalog/speaker_discovery*.go and historical_v11.go; internal/app/speaker_discovery.go; matching tests. Extend records/migration/snapshot/invalidation, runtime/CLI dispatch, versioned schemas/help, desktop bridge tests and public contract documentation. See research.md, data-model.md, contracts/runtime.md and quickstart.md.

## Workflow notes

No extension hooks are installed. The installed checklist prerequisite requires plan.md; setup_plan.py initialized this skeleton before the reviewer checklist was generated. No substantive plan preceded clarification/checklist. The actual Git branch, explicit feature directory and machine selector agree; helper BRANCH values report the feature basename rather than the codex/ branch prefix. Built-in requirements checklist is 16/16; the custom reviewer checklist remains 0/12 and its markers stay untouched under explicit autopilot authorization.
