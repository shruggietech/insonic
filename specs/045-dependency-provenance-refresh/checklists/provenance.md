# Provenance Requirements Checklist: Dependency refresh

**Purpose**: Reviewer requirements-quality prompts for S045.
**Created**: 2026-10-22
**Feature**: [spec.md](../spec.md)

**Review Ownership**: Reviewer-owned; checked markers mean requirements quality approval, not implementation completion.

## Scope and source integrity

- [ ] CHK001 Are all three original update dispositions and the replacement boundary specified? [Completeness, Spec FR-001, FR-008]
- [ ] CHK002 Are upstream checksums and local patch relationships required to remain consistent? [Consistency, Spec FR-003]
- [ ] CHK003 Is rejection of stale or missing source identity explicitly preserved? [Coverage, Spec FR-004]

## Qualification and handoff

- [ ] CHK004 Are platform, package and existing CLI coverage requirements complete? [Completeness, Spec FR-005]
- [ ] CHK005 Are preparation, qualification, total time and advisory/hard limits distinguished? [Clarity, Spec FR-006]
- [ ] CHK006 Are review-round, publication, cleanup and merge boundaries unambiguous? [Consistency, Spec FR-007, FR-009]

## Notes

These six prompts remain reviewer-owned. The separate built-in requirements checklist is author-validated. No extension hooks are installed.
