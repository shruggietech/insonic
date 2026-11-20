# Maintainer requirements quality checklist for S078

**Purpose** Review reproducibility, fallback and privacy requirements.
**Created** 2026-11-20
**Feature** [Specification](../spec.md)

Markers record reviewer approval of requirements quality. Autopilot leaves them
for the reviewer.

## Completeness and consistency

- [ ] CHK001 Are the versioned skill, two guides and single canonical opt-in specified together? [Completeness, FR-001/FR-002]
- [ ] CHK002 Does the scope protect the surrounding instructions and product contracts? [Consistency, FR-002/FR-005]
- [ ] CHK003 Are private materials excluded consistently from the repository and publication handoff? [Consistency, FR-004/FR-006]

## Scenario coverage and measurability

- [ ] CHK004 Are the existing-project lookup and unavailable-connection fallback both specified? [Coverage, FR-003/SC-002]
- [ ] CHK005 Can the maintainer distinguish documented fallback, tested connected behavior and pending checks? [Measurability, SC-002/SC-004]
