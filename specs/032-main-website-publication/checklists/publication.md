# Publication Requirements Checklist: Reliable website publishing from main

**Purpose**: Review requirement quality for publication identity, recovery and unattended evidence.

**Created**: 2026-10-15

**Feature**: [spec.md](../spec.md)

**Review Ownership**: Reviewer-owned requirements-quality artifact. Autopilot analysis records its evaluation separately; implementation does not modify markers.

**Marker Semantics**: `[x]` means reviewer approval of requirement quality, not implementation completion.

## Authority and identity

- [ ] CHK001 Are source qualification and trusted publication boundaries explicit? [Completeness, Spec FR-001, FR-010]
- [ ] CHK002 Are website-source and product-release identities distinguished? [Clarity, Spec FR-002, FR-003]
- [ ] CHK003 Are historical schema/path and destination configuration obligations consistent? [Consistency, Spec FR-004]

## Failure and recovery

- [ ] CHK004 Are concurrent updates, response loss and identical replay covered? [Coverage, Spec FR-005, FR-009]
- [ ] CHK005 Is successful deployment distinguished from failed live verification? [Clarity, Spec FR-006, FR-009]
- [ ] CHK006 Are deadlines, credential redaction and owned cleanup required? [Completeness, Spec FR-008]

## Evidence and scope

- [ ] CHK007 Are complete byte, download and latest-routing evidence specified? [Measurability, Spec FR-006]
- [ ] CHK008 Is the actual-live browser matrix explicit and distinguished from fixtures? [Clarity, Spec FR-007]
- [ ] CHK009 Are CI timing boundaries and required qualification outcomes stated? [Measurability, Spec FR-011, SC-004]
- [ ] CHK010 Are release/merge authorization and excluded application work explicit? [Coverage, Spec Assumptions]

## Notes

All markers remain reviewer-owned. Analysis may evaluate these criteria without changing them. No checklist item adds a new owner approval gate under the explicitly authorized autopilot scope.
