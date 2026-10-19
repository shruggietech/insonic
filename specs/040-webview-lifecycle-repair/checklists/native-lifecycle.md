# Native lifecycle requirements checklist

**Purpose**: Reviewer-owned requirements-quality assessment for [S040](../spec.md).
**Created**: 2026-10-19.
**Review ownership**: Unchecked means awaiting reviewer assessment; a checked marker would describe requirement quality, not implementation completion.

## Completeness and coverage

- [ ] CHK001 Are all five historical failure classes separately identified and retained? [Completeness, FR-002]
- [ ] CHK002 Does the scope cover both normal startup and all three actual native exit actions? [Coverage, US1/US2]
- [ ] CHK003 Are partial initialization, pending responses and zero-exit live descendants explicitly covered? [Edge Cases, FR-006/010/011]
- [ ] CHK004 Are actual runtime identity and merely available runtime identity distinguished in experiment design? [Clarity, FR-003]

## Evidence and constraints

- [ ] CHK005 Must every behavior change have a failing pre-correction regression and a corrected result? [Measurability, FR-005/SC-002]
- [ ] CHK006 Are predictions, controls, stopping rules and finite-sample limitations explicit? [Consistency, FR-009/SC-005]
- [ ] CHK007 Are original deadlines, natural release and exact source/package authority preserved? [Consistency, FR-004/013/014]
- [ ] CHK008 Are hidden execution, disposable native input and bounded content-free evidence defined? [Privacy and execution, FR-007/008]
- [ ] CHK009 Is the historical S039 exception excluded from S040 repair acceptance? [Acceptance, SC-004]
- [ ] CHK010 Are issue updates, unfinished claims and merge/release boundaries explicit? [Completeness, FR-012/015/016]
