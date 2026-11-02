# Tasks: Contextual speaker discovery and resolution

Input: spec.md, plan.md, research.md, data-model.md and contracts/runtime.md. S072 closes #227-#229 through one PR. TDD is required by autopilot; meaningful security, scope and recovery coverage is mandatory.

## Phase 1: Setup

- [x] T001 Verify slice identity, owner authorization, clean baseline and no extension hooks in specs/072-contextual-speaker-discovery/plan.md.
- [x] T002 Complete specification, clarification, design and review checklist in specs/072-contextual-speaker-discovery/.
- [x] T003 Complete cross-artifact analysis and record its gate in specs/072-contextual-speaker-discovery/verification.md; set existing GitHub Project issue items to S072/In progress without creating an issue.

## Phase 2: Foundation

- [x] T004 Capture immutable schema-11 DDL/digest in internal/catalog/historical_v11.go before modifying reflective records.
- [x] T005 Add failing versioned evidence/parser/resolution contract fixtures in internal/speakerdiscovery/discovery_test.go (self-introduction, mention, quotation, duplicates, conflicts, multilingual aliases, circular evidence).
- [x] T006 Define bounded evidence, coverage, current-source references, provider provenance and decision DTOs in internal/speakerdiscovery/types.go.

## Phase 3: US1 - Inspect grounded hypotheses (P1)

Independent test: current hypotheses preserve ambiguity and provenance; corrections atomically remove stale derived content; portable proof forgery and foreign references fail without changing mappings.

- [x] T007 [US1] Add failing SQLite/PostgreSQL discovery authority, isolation, lease/CAS/replay, invalidation and portable-state tests in internal/catalog/speaker_discovery_test.go and internal/catalog/postgres_test.go.
- [x] T008 [US1] Add typed aggregate, schema-12 migration across all historical paths and portable validation in internal/catalog/records.go, migration.go, jobs_migration.go and snapshot.go.
- [x] T009 [US1] Implement source-specific freeze, current inspection, atomic acceptance/proofs and invalidation in internal/catalog/speaker_discovery.go and exploration_outbox.go; exclude ordinary work writes from freshness.
- [x] T010 [US1] Verify populated v11 upgrade, current/foreign references, owner/source/alias/roster replacement, cancellation and tombstone replay in internal/catalog/speaker_discovery_test.go.

## Phase 4: US2 - Elected structured extraction (P1)

Independent test: an elected fixture provider yields fully validated current hypotheses; any malformed/foreign/oversized response, failed chunk, cancelled deadline or changed authority publishes nothing.

- [x] T011 [P] [US2] Add failing strict parser tests for duplicate keys, unknown fields, invented times/IDs, ambiguous attribution and exact citation validation in internal/speakerdiscovery/parse_test.go.
- [x] T012 [P] [US2] Add failing durable provider tests for local/hosted/Ollama routes, aggregate timeout/cancel, coverage, stale authority, lost acceptance and unchanged mappings in internal/app/speaker_discovery_test.go.
- [x] T013 [US2] Implement strict all-or-nothing parsing, reference validation, bounded observations and deterministic source-root deduplication in internal/speakerdiscovery/parse.go.
- [x] T014 [US2] Implement durable submission/worker/recovery, chunked elected provider generation and atomic publication in internal/app/speaker_discovery.go, app.go and work.go; reuse protected assistance transport.
- [x] T015 [US2] Add shared runtime/CLI human and JSON discover/hypotheses commands, schema-derived help and operation validation in internal/contracts/, cmd/insonic/ and schemas/v1.2.0/.

## Phase 5: US3 - Explainable reproducible resolution (P1)

Independent test: fixed evidence/policy gives identical sorted decisions; independent acoustic/context agreement, disagreement, duplicate aliases, unknown/unregistered names and owner correction fences produce the specified five outcomes.

- [x] T016 [P] [US3] Add failing outcome, policy-bound, dedup/overlap/cycle and contradiction precedence tests in internal/speakerdiscovery/resolve_test.go.
- [x] T017 [P] [US3] Add failing authenticated acoustic receipt/current dataset/independent lineage and stale profile tests in internal/catalog/speaker_discovery_acoustic_test.go.
- [x] T018 [US3] Implement deterministic candidate-preserving resolution and explicit reasons/alternatives in internal/speakerdiscovery/resolve.go; never use confidence alone or duplicate roots as proof.
- [x] T019 [US3] Implement authenticated current acoustic evidence reader with mapping-transition normalization and independent enrollment validation in internal/catalog/speaker_discovery_acoustic.go; never call ExecuteMatch.
- [x] T020 [US3] Implement atomic expected-revision resolution and shared runtime/CLI resolve operation in internal/catalog/speaker_discovery.go, internal/app/speaker_discovery.go, internal/contracts/ and cmd/insonic/.
- [x] T021 [US3] Verify concurrent corrections, policy retries, receipt tampering, no resurrection and unchanged mappings/profile associations in internal/catalog/ and internal/app/ discovery tests.

## Phase 6: Contract alignment and delivery

- [x] T022 [P] Align versioned aggregate/runtime/master/catalog schemas and help with desktop Operate bridge tests in schemas/v1.2.0/, internal/contracts/ and desktop/.
- [x] T023 [P] Document authoritative evidence, routing, outcomes, limits, uncertainty and advanced GUI lag in docs/v1.2.0/{speakers,cli,schema}.md; record feature and dated architecture decisions in CHANGELOG.md.
- [x] T024 Run quickstart fixtures, npm run check, npm test, Go test/vet, Python tests and affected site/frontend checks; inspect UTF-8/LF/mojibake and record exact results in specs/072-contextual-speaker-discovery/verification.md.
- [x] T025 Run Spec Kit converge against spec/plan/tasks and fix all material gaps; preserve reviewer checklist ownership in specs/072-contextual-speaker-discovery/checklists/evidence.md.
- [ ] T026 Commit and automatically push codex/072-contextual-speaker-discovery, open an official PR closing #227-#229, attach it, and move existing Project items to In review; record delivery references in specs/072-contextual-speaker-discovery/verification.md or the PR acceptance receipt.
- [ ] T027 Resolve every review finding within at most two rounds, verify exact-head full CI, retain terminal receipts outside workflow artifacts, and hand the PR to the owner for final review/merge; close delivery evidence in the PR acceptance receipt without a self-referential acceptance commit.

## Dependencies and implementation strategy

T001-T003 -> T004-T006 -> US1 -> US2 -> US3 -> integration/delivery. Pure parser and resolution test preparation can run independently after DTO design, but production acceptance depends on US1. Within each story write failing tests, implement, then run focused checks. US1 is the first verifiable increment, not a scope reduction: all three stories ship together.

Parallel examples: US1 migration test fixtures and proof fixture preparation use separate test files after T006; US2 T011/T012 and US3 T016/T017 may run independently. T022/T023 are independent after final interfaces. No implementation delegation is required.

## Coverage

FR001-003: T005-T010,T016-T019. FR004-006: T011-T015. FR007-010: T016-T021. FR011: T014-T015,T020,T022. FR012-013: T007-T010,T017,T021. FR014: T022-T025. FR015: T026-T027. SC001-005: T010-T012,T016-T017,T021,T024,T027. All twelve story acceptance scenarios have fixture or end-to-end coverage tasks. Existing issues remain the primary tracker; no new tracking issue is permitted.

## Phase 7: Convergence

- [x] T028 Preserve valid current manual mappings to inactive identities without adding those identities to provider candidate context; include their current identity authority in freshness and add regression coverage in internal/catalog/speaker_discovery.go, internal/speakerdiscovery/resolve.go and discovery tests per FR-010, FR-012 and US3/AC3 (partial, HIGH).
