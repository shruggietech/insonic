# Setup Requirements Checklist

Purpose: PR reviewer requirements-quality review. Created: 2026-10-13.

Ownership: reviewers own these markers; checked means requirement clarity, not completed implementation. Autopilot proceeds under the owner's explicit end-to-end instruction; this checklist remains a review artifact.

## Completeness and consistency

- [ ] CHK001 Are all five stages and optional navigation defined? [Spec FR-002]
- [ ] CHK002 Is setup absence explicitly independent from ordinary import? [Spec FR-001, FR-008]
- [ ] CHK003 Are accepted configuration and journey metadata distinct? [Spec FR-006, FR-007]
- [ ] CHK004 Are pre-workspace identity and retry requirements explicit? [Spec US2, FR-005]

## Failure, security and accessibility

- [ ] CHK005 Are stale writes and lost response corrections defined? [Spec FR-005]
- [ ] CHK006 Are secret values excluded from both draft domains? [Spec FR-009]
- [ ] CHK007 Are cancellation effects distinguished from work cancellation? [Spec FR-011]
- [ ] CHK008 Are focus, keyboard, progress and scaling requirements specified? [Spec FR-012]
- [ ] CHK009 Is configured readiness distinguished from authentication and inference? [Spec FR-010]
- [ ] CHK010 Are source/version documentation boundaries explicit? [Spec FR-014, Assumptions]

## Notes

Implementation evidence belongs in verification.md; this artifact does not mark implementation completion.
