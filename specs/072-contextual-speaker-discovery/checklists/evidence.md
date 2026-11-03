# Evidence and identity requirements checklist: Contextual speaker discovery

**Purpose**: Review provenance, independence, current authority and unattended provider boundaries in S072 requirements.

**Created**: 2026-11-02

**Feature**: [spec.md](../spec.md)

**Review Ownership**: Reviewer-owned requirements-quality checklist; [x] means reviewed requirement quality, not implementation completion. Autopilot may proceed under the owner's explicit authorization without changing these markers.

## Completeness and clarity

- [ ] CHK001 Are observations, local voices, identity hypotheses and accepted mappings explicitly distinguished? [Completeness, Spec FR-001, FR-002]
- [ ] CHK002 Are independent, derived and circular evidence relationships defined without a manual-only assumption? [Clarity, Spec FR-002, FR-003, FR-009]
- [ ] CHK003 Are multilingual names, equal spellings, quotes, mentions and abstention covered without automatic identity merging? [Coverage, Spec FR-006, FR-007]
- [ ] CHK004 Are all five resolution outcomes and their explanations measurable and distinct? [Clarity, Spec FR-008, SC-001]

## Integrity and failure coverage

- [ ] CHK005 Are current source references authoritative over provider-created IDs, intervals and claims? [Consistency, Spec FR-005]
- [ ] CHK006 Are concurrent corrections and source, roster, alias or profile changes addressed at publication and consumption? [Coverage, Spec FR-010, Edge Cases]
- [ ] CHK007 Are provider scope, routing, credentials, aggregate budgets and cancellation requirements explicit? [Completeness, Spec FR-004, FR-013]
- [ ] CHK008 Are workspace isolation, retry/unknown outcomes and portable-state recovery covered across supported backends? [Coverage, Spec FR-013, SC-003]
- [ ] CHK009 Are preserved mappings/profiles/source text and invalidation diagnostics consistent across failure and success? [Consistency, Spec FR-010, FR-012]

## Delivery boundaries

- [ ] CHK010 Are the minimal shared CLI journey and later enrollment/application workflow boundaries explicit? [Completeness, Spec FR-011, FR-012, Assumptions]
- [ ] CHK011 Are deterministic contract checks separated from model-quality and acoustic-accuracy claims? [Measurability, Spec FR-014, SC-001]
- [ ] CHK012 Are existing-issue closure, two-round review limit and owner merge boundary explicit? [Completeness, Spec FR-015, SC-005]

## Notes

All twelve items are reviewer-owned and remain unchecked. The built-in requirements checklist is a separate author-maintained artifact. No extension hooks are installed. Scope/depth defaults are standard PR review focused on evidence integrity and provider/current-authority boundaries; no unresolved question requires owner input.
