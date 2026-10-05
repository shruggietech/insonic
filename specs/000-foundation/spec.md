# Feature specification: repository foundation

**Slice**: S000 (`000-foundation`)\
**Created**: 2026-10-04\
**Status**: Local foundation complete; upstream proof deviation recorded\
**Tracking**: Local foundation record; application slices use GitHub Issues and the Project

## User scenarios and acceptance

### Understand the intended system (P1)

The owner can read a divided v0.0.0 specification and follow a staged path from a local CLI library to GUI querying, timeline and saved graph views.

Acceptance: every owner requirement appears in the stable map; technology choices are justified; implemented and future behavior are distinguished; diagrams read top-down; public prose contains no private design-source or case-specific information.

The 2026-10-04 amendment includes library-wide speaker audio segments, elected voice-model training and retrieval, early metadata capture, origination-date/timezone inputs, CLI-first parity with documented experimental GUI lag, and complete filesystem/S3, SQLite/PostgreSQL and LadybugDB/ArcadeDB adapter contracts.

### Maintain upstream branding (P1)

A maintainer can manually compare the formal kit with upstream, adopt a newer verified kit and recover the old integration on failure.

Acceptance: themed README assets are exact upstream bytes; checks are read-only; updates validate checksums, inventory and paths; failures restore kit and receipt; offline verification works.

### Prepare sustainable repository operations (P2)

Contributors have a compact README, changelog, license, initialized Spec Kit, guidance/templates, efficient CI and concrete issue/milestone/project setup for publication.

Acceptance: documentation compiles as static Next.js with local diagrams/assets; release highlights end with the changelog link; human-approved squash and automatic branch deletion are prepared. Publication follows separate owner authorization.

## Requirements

- FR-001: Author the complete intended system in `docs/v0.0.0/` with navigation and requirement IDs.
- FR-002: Integrate the upstream-owned formal brand and manual updater.
- FR-003: Initialize actual Spec Kit infrastructure and constitution.
- FR-004: Prepare public files, targeted CI and GitHub tracking/settings tooling.
- FR-005: Build static documentation suitable for release-matched offline help.
- FR-006: Save UTF-8 without BOM, check corruption and hide Windows console children.
- FR-007: Preserve full product scope while recording real dependency limitations.
- FR-008: Define speaker segment, dataset, training-run and model-version lineage around stable speaker identities.
- FR-009: Define early immutable metadata capture and explicit date/timezone selection without a per-item review queue.
- FR-010: Require portable storage/catalog/graph adapters with full officially supported alternate-backend behavior.
- FR-011: Keep CLI-first architecture fixed and minimize repetitive approvals with saved, revisioned policies.
- FR-012: Provide described, example-rich JSON contracts under a release-matched master schema and generate the documentation reference from them.
- FR-013: Keep public documentation authoritative and free of slice/history prose outside the changelog; refresh the linked glossary at every major release.
- FR-014: Provide a branded product landing page at the public root and exclude it from the installed GUI help export.

## Scope and edge cases

The original local foundation scope excludes media processing, model downloads, native product packages and repository publication. Hosted setup and first-push evidence are recorded separately after owner-authorized execution; native product behavior requires its application slices.

An unavailable upstream release does not mutate the brand. Immutable-version drift is an error. Failed replacement restores kit and receipt. Documentation/schema-version drift fails checks. Cueson's SRT/WebVTT zero-cue limitation remains an S007 compatibility task; ASS/SSA behavior is accounted for separately.

## Success criteria

Project integrity checks, focused maintainer tests and the static build pass. Site links resolve locally and every v0.0.0 page is present. The real upstream import verifies its inventory. The owner can identify the next slice and acceptance without private external material.
