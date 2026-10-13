# Distribution Requirements Quality Checklist

**Purpose**: PR reviewer assessment of platform retirement and installation ownership requirements.

**Created**: 2026-10-13

**Feature**: [spec.md](../spec.md)

Checkboxes are reviewer-owned requirements-quality approval, not implementation status.

## Completeness and consistency

- [ ] CHK001 Are supported targets and all four required variants explicit? [Completeness, Spec FR-001/004]
- [ ] CHK002 Are current and historical release obligations distinguished? [Consistency, Spec FR-004/006]
- [ ] CHK003 Are obsolete repository checks and all owned runtime/build surfaces included? [Coverage, Spec FR-002/003]
- [ ] CHK004 Is remaining qualification preserved without weakened integrity? [Consistency, Spec FR-007]

## Installation boundaries

- [ ] CHK005 Are package inclusion, architecture and feature-specific prerequisites explicit? [Clarity, Spec FR-008/009]
- [ ] CHK006 Are alias containment and existing-target rejection defined before copying? [Coverage, Spec FR-011]
- [ ] CHK007 Are cleanup ownership, target substitution and original-error preservation defined? [Coverage, Spec FR-012]
- [ ] CHK008 Are update/rollback/removal effects separated from MSI and user data? [Consistency, Spec FR-013]

## Notes

The specification supplies every criterion. Autopilot proceeds under existing owner authorization; these review markers do not create an attended pre-release gate.
