# Tasks: S035 HTTP evidence extraction and provider diagnostics

## Phase 1: Setup

- [x] T001 Verify three issue IDs, branch and feature directory; author specs/035-http-adapter-contracts/spec.md and requirements checklist.
- [x] T002 Research extraction/provider boundaries and generate plan.md, research.md, data-model.md, contracts/providers.md and quickstart.md in specs/035-http-adapter-contracts.

## Phase 2: Foundation

- [x] T003 Add reason-vocabulary/schema and redaction regressions, then align locally generated reasons in internal/contracts/operator.go, contracts.go and schemas/v1.0.1/runtime-response.schema.json (FR-003..FR-006).

## Phase 3: US1 Reach HTTP extraction (P1)

Independent test: both identifiers pass schema and reach an owned provider; invalid citations and spoofed clocks cannot publish.

- [x] T004 [US1] Add failing public-schema/normalization tests and app current-chunk/foreign/unknown-field/clock-spoof regressions in internal/evidence and internal/app/explore_http_test.go (FR-001/FR-002, SC-001).
- [x] T005 [US1] Canonicalize legacy http in internal/evidence/evidence.go, align extraction enum in schemas/v1.0.1/runtime-request.schema.json and preserve strict response shape/duplicate handling in internal/evidence/http.go (FR-001/FR-002/FR-005).

## Phase 4: US2 Provider failures (P2)

Independent test: both adapters pass the complete status/protocol/body/transport/deadline/cancel matrix without exposing fixture secrets.

- [x] T006 [US2] Add failing status/protocol/body/transport/deadline/cancel matrix in internal/assistance and app propagation regressions (FR-003/FR-005, SC-002/SC-004).
- [x] T007 [US2] Correct generic HTTP and Ollama boundary classification in internal/assistance/http.go, ollama.go and internal/app/assistance.go while preserving compatible reasons and route guards (FR-003/FR-005/FR-006).

## Phase 5: US3 Actual output bounds (P2)

Independent test: artifact bytes, native rows/bytes and provider bytes retain output_limit with their own reason and correction.

- [x] T008 [US3] Add failing materialization/native/provider reason regressions and correct internal/artifact/service.go, internal/graph/types.go and assistance provider-proposal byte ceilings in internal/app/assistance.go (FR-004, SC-003).

## Phase 6: Delivery

- [x] T009 Add deterministic provider/citation/limit installed CLI fixture helper under scripts/ and integrate scripts/cli_contract_qualification.py for exact Windows/Linux packages (FR-007, SC-001..SC-004).
- [x] T010 Update docs/v1.0.1/graph.md and CHANGELOG.md; preserve public/internal separation and record compatibility choices (FR-006).
- [x] T011 Execute regression-first, full Go/vet, root npm/Python and affected site/frontend/package checks; record evidence in specs/035-http-adapter-contracts/verification.md and converge (FR-001..FR-007, SC-001..SC-005).
- [ ] T012 Commit and publish the three-issue PR with milestone/Project tracking; resolve at most two review rounds, verify exact-head CI and every thread, stop before merge (SC-005).

## Dependencies and execution strategy

T001 -> T002 -> read-only analysis gate -> T003 -> T004/T005 -> T006/T007 -> T008 -> T009/T010 -> T011 -> T012. US1 is the first independently useful result; all three issues remain required. Independent tests may run in parallel through hidden launchers, but shared-file edits remain sequential. Read-only research agents inspected the two protocol boundaries; implementation is owned by the primary agent.
