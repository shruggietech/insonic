# Release Requirements Checklist: v1.1.0 preparation

**Purpose**: Reviewer assessment of release, historical and qualification requirement quality.
**Created**: 2026-10-31
**Feature**: [spec.md](../spec.md)

**Review Ownership**: Reviewer-owned requirements-quality artifact. An `[x]` records reviewer assessment, not implementation completion. Generated items remain unchecked; the existing owner autopilot authorization permits implementation to proceed without an additional approval gate.

## Completeness and clarity

- [ ] CHK001 Are all coordinated version authorities and candidate status explicitly covered? [Completeness, Spec FR-001, FR-004]
- [ ] CHK002 Is preservation of original assets and historical schema/document authority distinguished from carrying current fixes forward? [Clarity, Spec FR-003]
- [ ] CHK003 Are genuine capture identity, byte integrity and isolated desktop requirements explicit? [Completeness, Spec FR-005]
- [ ] CHK004 Is the complete supported package, installer, native and backend acceptance scope defined? [Completeness, Spec FR-006]

## Consistency and measurable acceptance

- [ ] CHK005 Are release-version, catalog-version and upstream dependency identities distinguished? [Consistency, Spec FR-001, FR-008]
- [ ] CHK006 Are branch head, temporary merge revision and later accepted-main identities separated? [Clarity, Spec FR-006, FR-010]
- [ ] CHK007 Are readiness, separate timings, advisory target, hard limit and provisional reserve unambiguous? [Measurability, Spec FR-007]
- [ ] CHK008 Are mandatory checks independent of timing acceptance and evidence retained before reruns? [Consistency, Spec FR-007]

## Recovery, authority and limitations

- [ ] CHK009 Are missing, corrupt, expired or wrong-version capture/acceptance artifacts covered without substitute evidence? [Coverage, Spec Edge Cases, FR-005, FR-007]
- [ ] CHK010 Does the review/publication boundary specify exactly two rounds and owner-controlled merge/release? [Clarity, Spec FR-009]
- [ ] CHK011 Are real-model CPU/NVIDIA and acoustic claims separated from deterministic qualification? [Clarity, Spec FR-011]
- [ ] CHK012 Does the continuation preserve the exact accepted candidate bytes after qualification? [Coverage, Spec FR-010, SC-005]

## Notes

Standard-depth reviewer checklist focused on release authority and qualification evidence. The built-in requirements checklist is separately maintained and passes. The checklist helper requires plan.md even before the protocol's planning phase, so the installed template resolver supplied the checklist template; prerequisites are rechecked after setup-plan. No extension hooks exist.
