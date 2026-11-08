# Tasks: S076 answer and startup contract reliability

**Input**: spec.md, plan.md, research.md, data-model.md, contracts/reliability.md.

## Phase 1: Setup

- [x] T001 Read governing contracts, current issues and autopilot rules; create specs/076-answer-startup-reliability/spec.md and checklists.
- [x] T002 Research and document schema/error-origin decisions in specs/076-answer-startup-reliability/plan.md and supporting design artifacts.

## Phase 2: Foundation

- [x] T003 Complete blocking analysis and verify hidden toolchain/prerequisite readiness; retain .local/s076 preparation evidence.

## Phase 3: User Story 1 (P1)

**Goal**: Correct typed native answers and actionable bounded repair without weakening source authority.

**Independent test**: Reported two-source Monday reproduction plus unchanged seven-case model campaign.

- [x] T004 [US1] Add failing schema/type and native-format/repair tests in schemas/answer_synthesis_test.go, internal/assistance/ollama_test.go and internal/app/answer_synthesis_test.go (FR-001/FR-002/FR-003, SC-002).
- [x] T005 [US1] Implement self-contained exact-version/package synthesis schema in schemas/answer_synthesis.go, preserving reference definitions and at most 128 claims (FR-001).
- [x] T006 [US1] Wire raw shape validation, at most one generated repair detail and operation-specific local Ollama schema format in internal/answers/synthesis.go, internal/app/answer_synthesis.go and internal/assistance/ollama.go; retain zero-or-one repairs and shared provider deadline (FR-002/FR-003/FR-004).
- [x] T007 [US1] Verify adversarial reference/quote/source/cancellation and other adapter regressions in internal/answers, internal/app and internal/assistance (FR-004, SC-002).
- [ ] T008 [US1] Run exact-model installed CLI evaluation through scripts/qualify-answers.py; retain all seven scored cases, model/corpus/source/executable identity and cleanup in specs/076-answer-startup-reliability/evidence (FR-005, SC-001).

## Phase 4: User Story 2 (P2)

**Goal**: Preserve catalog incompatibility and data through cold startup and maintenance.

**Independent test**: Valid control then closed disposable catalog marker 999, cold show and migrate exit 3, unchanged authority.

- [x] T009 [US2] Add failing origin/non-mutation and startup report tests in internal/catalog/migration_test.go and internal/runtime/startup_test.go (FR-006/FR-007, SC-003).
- [x] T010 [US2] Tag schema incompatibility origins in internal/catalog/migration.go, add generated reason in internal/contracts/operator.go and current response schema, preserve it in internal/runtime/runtime.go (FR-006/FR-007).
- [ ] T011 [US2] Add and execute maintained installed CLI cold/valid/maintenance checks using isolated fixtures in scripts/cli_operator_qualification.py or a focused qualification helper/tests (FR-007, SC-003).

## Phase 5: Qualification and delivery

- [x] T012 Prepare matching v1.3.1 authorities and public answer/startup documentation using scripts/release.py; record changes/decisions in CHANGELOG.md and internal highlights (FR-008).
- [ ] T013 Run npm run check, npm test, Go tests/vet and affected Python/site/frontend/build checks with retained readiness/timing evidence and format/encoding sanity checks (FR-009, SC-004).
- [ ] T014 Converge against spec/plan/tasks and retain compact audited verification in specs/076-answer-startup-reliability/verification.md and evidence (FR-001..FR-009).
- [ ] T015 Commit, push and open one official issue-linked PR; associate milestone/Project metadata and retain review/CI status outside Actions (.local/s076 and PR) (FR-009).
- [ ] T016 Address every received review comment, request no more than one second round, retain exact-head green terminal CI and hand off for owner merge (FR-009, SC-005).

## Dependencies and execution order

T001 -> T002 -> T003 -> US1 and US2 regressions/implementation -> matching version preparation -> package and actual-model qualification -> convergence -> official PR -> exact-head review/CI handoff. T008 needs the implemented candidate package. T010's current schema reason applies after T012 copies the next-version schema.

The two stories are independently testable. Tests in separate schema/adapter files can be prepared independently, as can catalog origin and report tests. Research was delegated under speckit-plan; implementation is serial in the existing checkout to avoid shared-file contention. Deliver both stories together, without an intermediate user gate.
