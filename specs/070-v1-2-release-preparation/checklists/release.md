# Release Checklist: v1.2.0 preparation

**Purpose**: Reviewer requirements-quality assessment of release provenance and scope.
**Created**: 2026-11-02
**Feature**: [spec.md](../spec.md)

**Review Ownership**: Reviewer-owned; [x] means requirements quality is satisfied, not implementation completion. Implement reads markers without modifying them. Owner autopilot already authorizes continuation; these advisory reviewer items do not add an approval gate.

## Completeness and clarity

- [ ] CHK001 Are candidate and historical version authorities distinguished? [Clarity, Spec FR-001]
- [ ] CHK002 Is dependency checksum preservation explicit? [Completeness, Spec FR-002]
- [ ] CHK003 Are documentation, highlights and CLI/GUI differences covered? [Coverage, Spec FR-003]
- [ ] CHK004 Is the complete supported package and backend matrix specified? [Completeness, Spec FR-004]
- [ ] CHK005 Is current capture provenance distinguished from historical screenshots? [Clarity, Spec FR-005]

## Acceptance and exceptions

- [ ] CHK006 Are timing targets, hard limits and preparation boundaries measurable? [Measurability, Spec FR-006]
- [ ] CHK007 Are final source identity, review rounds and durable evidence requirements explicit? [Completeness, Spec FR-007]
- [ ] CHK008 Are merge, publication and tracking boundaries unambiguous? [Clarity, Spec FR-008]
- [ ] CHK009 Are unelected real-model checks and later publication qualification distinguished? [Consistency, Spec FR-009]
- [ ] CHK010 Are checksum failures, stale captures and changed-head reruns addressed? [Coverage, Spec Edge Cases]

## Notes

Standard-depth PR reviewer checklist. Created with ten unchecked requirements-quality items; no additional owner question is needed.
