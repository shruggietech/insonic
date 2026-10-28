# Tasks: Import interruption recovery

**Input**: spec.md, plan.md, research.md, data-model.md, contracts/recovery-receipt.md and quickstart.md.
**Tests**: Meaningful failing-first integrity, ownership and recovery regressions are required.

## Phase 1: Setup

- [x] T001 Align branch, selector and specs/057-import-interruption-recovery/; complete specify/clarify and requirements.md with owner-approved S057/#147 scope.
- [x] T002 Inspect current admission, source/package prerequisites and ignore patterns; validate an owned hidden Windows file-open prototype and record decisions in research.md (FR-001, FR-005, FR-006).

## Phase 2: Foundational

- [x] T003 Add real blocked-open/release and refusal/lifetime regressions in tests/file_open_barrier_test.py before scripts/file_open_barrier.py implementation (FR-001, FR-006, FR-007; SC-003).
- [x] T004 Implement hidden owned Windows RW oplock and Linux lease-helper barriers in scripts/file_open_barrier.py with validated ACK/break lifetime, original kernel/enclosing bounds and joined cleanup (FR-001, FR-006, FR-007, FR-009).

## Phase 3: US1 - Recover an interrupted import

**Independent check**: An actual runtime dies after complete report publication and a distinct replacement admits one recording with original candidate bytes/provenance.

- [x] T005 [US1] Add failing-first current-admission native tests in internal/library/admission_recovery_test.go proving complete candidate/provenance reuse and duplicate-free replay (FR-002, FR-004; SC-001).
- [x] T006 [US1] Implement complete-pair source/artifact verification and reuse in internal/library/admission_recovery.go and internal/library/admission.go, preserving partial/observed-failure retirement behavior and existing fences (FR-002, FR-004, FR-006).
- [x] T007 [US1] Implement the actual same-source runtime interruption/restart campaign in scripts/cli_recovery_qualification.py using existing media-tool configuration, independent catalog/byte reads, ordinary attempt recovery and owned cleanup (FR-001, FR-002, FR-004, FR-007).

## Phase 4: US2 - Refuse mismatched recovered metadata

**Independent check**: Equal-length changed input and report corruption refuse stale admission after real runtime replacement.

- [x] T008 [US2] Add changed-source, corrupted-report/canonical, partial candidate and current-reference/fencing negatives in internal/library/admission_recovery_test.go and tests/cli_recovery_qualification_test.py (FR-003, FR-004; SC-002, SC-003).
- [x] T009 [US2] Complete negative campaign scenarios and proportional recovery corrections in scripts/cli_recovery_qualification.py and internal/library/admission_recovery.go; preserve unrelated accepted authority (FR-003, FR-004, FR-007).

## Phase 5: US3 - Trust repeatable qualification

**Independent check**: All six image/package contexts provide complete validated evidence; omission/mutation fails without retroactively requiring new evidence from immutable releases.

- [x] T010 [US3] Add strict missing/altered/stale identity, boundary, process, attempt, outcome and cleanup receipt tests in tests/cli_recovery_qualification_test.py and tests/release_test.py, including exact original-v1.0.2 compatibility and changed/unknown receipt refusal (FR-008, FR-009; SC-003).
- [x] T011 [US3] Integrate recovery receipts into scripts/cli_contract_qualification.py, scripts/qualify.py, tests/fixtures/cli-corpus.json and modern scripts/release.py acceptance; pin only immutable original-v1.0.2 package receipt hashes in scripts/release_history.py and update shared release fixtures (FR-005, FR-008, FR-009; SC-001, SC-004).

## Phase 6: Verification and delivery

- [x] T012 Update docs/v1.0.2/storage.md, docs/v1.0.2/development.md and CHANGELOG.md with recovery/qualification contracts, explicit limits and dated tooling decisions (FR-002, FR-006, FR-008, FR-009).
- [x] T013 Run focused/full product/Python checks, npm run check, npm test and affected site tests/build; analyze/converge every FR/SC and retain concise results in specs/057-import-interruption-recovery/verification.md (all requirements).
- [x] T014 Create S057 milestone/Project linkage, commit/push/open and attach the official PR referencing #147, monitor exact-head CI with independent retained audits, resolve all findings through at most two Codex rounds, and stop before merging (SC-004).

## Dependencies and execution strategy

Setup -> barrier foundation -> US1 -> US2 -> US3 -> verification/delivery. US2's service negatives can be added with US1 tests before the shared recovery implementation. Receipt validation tests precede integration. Edits sharing admission.go/qualification/release files remain sequential. The smallest useful increment is verified complete-pair recovery, delivered with actual process and negative evidence before PR handoff.

Independent read-only research and distinct native/Python focused suites may run concurrently. Spec Kit directed one research agent; no parallel implementation agents are required. No custom checklist is needed beyond the complete built-in requirements-quality checklist for this bounded slice.
