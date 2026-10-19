# Native Lifecycle Requirements Quality Checklist

**Purpose**: Reviewer assessment of diagnostic and ownership requirement quality.

**Created**: 2026-10-19

**Feature**: [Specification](../spec.md)

Reviewer-owned: checked items mean requirements-quality approval, not implementation completion. Implementation does not toggle these markers.

## Diagnostic Completeness

- [ ] CHK001 Are both output streams and result classifications specified separately from successful machine output? [Completeness, Spec FR-001]
- [ ] CHK002 Is the distinction between observed failure, hypothesis and demonstrated cause explicit? [Clarity, Spec FR-003]

## Ownership and Lifecycle

- [ ] CHK003 Are profile, descendant-process and cleanup ownership boundaries explicit? [Coverage, Spec FR-005]
- [ ] CHK004 Are repeated cold/warm and workspace/no-workspace cases included without permitting native-action substitutes? [Consistency, Spec FR-004]

## Delivery and Verification

- [ ] CHK005 Are complete proof identity and rejection requirements consistent with diagnostic failure handling? [Consistency, Spec FR-006]
- [ ] CHK006 Are the 660-second budget, bounded review rounds and unmerged handoff measurable? [Measurability, Spec FR-007/FR-008]

## Notes

Six reviewer-owned questions remain separate from the built-in spec quality checklist; owner autopilot already authorizes implementation within this scope.
