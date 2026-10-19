# Tasks: CLI contract and operator-feedback correctness

**Input**: [Specification](spec.md), [plan](plan.md), research, data model and operator contract.

## Phase 1: Setup

- [x] T001 Verify merged baseline, matching S039 branch/directory, constitution and issue mapping in specs/039-cli-contract-correctness/spec.md.
- [x] T002 Complete specify/clarify/checklist/plan research and design artifacts in specs/039-cli-contract-correctness/.

## Phase 2: Foundation

- [x] T003 Complete blocking cross-artifact analyze and read-only checklist status review in specs/039-cli-contract-correctness/verification.md before implementation (FR-001..018).
- [x] T004 Establish hidden noninteractive test execution, existing ignore/toolchain checks and regression ownership in specs/039-cli-contract-correctness/verification.md (FR-015..018).

## Phase 3: US1, choose and execute valid commands

**Independent test**: Enumerate accepted registry/alias inventory and execute actual emitted examples against controlled prerequisites.

- [x] T005 [P] [US1] Add failing command/option/operand completeness and actual-example parser/semantic tests in cmd/insonic/commands_test.go and tests/cli_help_examples_test.py (FR-001..004).
- [x] T006 [US1] Add reviewed purposes, meanings/defaults/conditions/exclusions, enums/cursors, backup operands/ownership and valid example tokens in cmd/insonic/commands.go and command guidance metadata (FR-001..004).
- [x] T007 [US1] Correct positional JSON/session/vault templates, unusable options, mapping cursor and pyannote example contracts in schemas/cli_help.go and schemas/v1.0.1/runtime-request.schema.json (FR-002..004).
- [x] T008 [US1] Execute emitted examples with an explicit ledger and only documented substitutions using scripts/cli_contract_qualification.py and existing installed fixture modules (FR-004/016; SC-001/003).
- [x] T009 [P] [US1] Add failing persisted migration/boundary/replay/restore tests for exactly 1..600000 milliseconds in internal/catalog/ and schemas/registry_test.go (FR-007/017).
- [x] T010 [US1] Freeze schema-9 authority, migrate SQLite/PostgreSQL checks forward, preserve schema-9 snapshots and align request bounds in internal/catalog/, schemas/v1.0.1/catalog-snapshot.schema.json and runtime-request.schema.json (FR-007/017).

## Phase 4: US2, understand and correct rejections

**Independent test**: Installed malformed/Unicode/destination matrix distinguishes stable causes, preserves exits and discloses no private values.

- [x] T011 [P] [US2] Add failing malformed/duplicate/shape/enum/version/protected-input diagnostics and destination tests in schemas/, internal/contracts/, internal/credentialcmd/ and cmd/insonic/ (FR-005/006/015).
- [x] T012 [US2] Generate safe structured reasons/known constraints/locations across schema, parser, credential and runtime boundaries in schemas/registry.go, internal/contracts/, internal/credentialcmd/ and cmd/insonic/ (FR-005/015/017).
- [x] T013 [US2] Classify missing workspace, existing recording destination and catalog output existence/parent/permission/directory/write failures in cmd/insonic/main.go and relevant runtime export handlers (FR-006).
- [x] T014 [P] [US2] Add multilingual exact-boundary/invalid-UTF8 tests and use Unicode code point counts without normalization in internal/catalog/identity.go, internal/models/manifest.go and other public maxLength validators (FR-008/017).
- [x] T015 [US2] Preserve blank/multiline model identity bytes and make identity displays visibly escaped in cmd/insonic/human.go; document intentional preservation in docs/v1.0.1/models.md (FR-009).

## Phase 5: US3, trust observed results and bounded summaries

**Independent test**: Warm/cold credential and typed-result failures, rich transcripts/readiness/search and capture byte pages retain all conclusions and exact machine authority.

- [x] T016 [P] [US3] Add failing bootstrap/zero-receipt/legitimate-partial regressions in cmd/insonic/ and internal/app/ (FR-010/011/015).
- [x] T017 [US3] Construct success only after bootstrap and omit only unestablished typed results before view conversion in cmd/insonic/main.go, setup.go and internal/app/ handlers (FR-010/011).
- [x] T018 [P] [US3] Add failing bounded identity/task/correction/verbosity and all capture-section/multibyte-page tests in cmd/insonic/human_test.go (FR-012..015).
- [x] T019 [US3] Prioritize complete observed summaries, exact retrieval commands, setup CLI follow-ups, quoted model identities and capture UTF-8 interior/authority in cmd/insonic/human.go (FR-009/012..015).

## Phase 6: Integration and delivery

- [x] T020 Integrate installed regressions for all eleven issues, semantic example coverage, UTF-8/LF/no-BOM/mojibake checks and owned cleanup in scripts/ and specs/039-cli-contract-correctness/verification.md (FR-004/015/016; SC-001..004).
- [x] T021 Update authoritative CLI/model/ingestion/catalog/schema docs, CHANGELOG.md and GitHub milestone/Project mapping for eleven issues; retain #114/#147 open in docs/v1.0.1/, CHANGELOG.md and GitHub (FR-017/018).
- [x] T022 Run required root, Go/vet/native/backend, Python, frontend/site and installed-package checks; converge, commit, push and open/attach official PR; satisfy exact-head CI <=660 seconds and at most two review rounds with every thread resolved; stop unmerged and record final evidence in specs/039-cli-contract-correctness/verification.md (FR-018; SC-005).

## Dependencies and parallel execution

T001-T004 precede product edits. Story regressions precede their corresponding implementation. After foundation, help/example metadata, catalog/Unicode work and rendering/observed-results work can run in disjoint files. Parent owns generated diagnostics/export classification and integration. Coordinate main.go and current runtime-request schema edits sequentially; never overwrite another worker's changes. T020-T022 require all stories.

US1 example: command-registry regressions and catalog migration regressions run in parallel; implementation follows each failing regression. US2 example: schema-diagnostic and catalog/model Unicode tests use disjoint files. US3 example: typed-observation tests and human-page tests are disjoint. All three independently testable stories are required for this approved slice; an MVP subset is not completion.

## Phase 7: CI remediation

- [x] T023 Separate cold native compilation from full native acceptance in scripts/qualify.py, retain both 180-second child allowances and all six package/tag assertions, verify failure cannot publish a receipt and keep the elected hosted path within 660 seconds (FR-018; SC-005).

- [x] T024 Preserve non-missing workspace-open failures, classify only genuinely missing roots/configs as workspace_missing and add real CLI/installed damaged-configuration regressions for review finding PRRT_kwDOU8_K3s6t1dFK (FR-005/006/015).

- [x] T025 Overlap Windows proof-consumer setup with assembly, await the successful exact-run/revision/attempt assembly and immutable artifact before download, retain all four proof scopes and final verification, and measure the complete elected path within 660 seconds (FR-018; SC-005).

- [x] T026 Overlap candidate tool preparation and qualified Linux/docs downloads with Windows completion, require both successful exact-run Windows package producers and matching artifacts before collection, and retain the complete exact-source candidate checks (FR-018; SC-005).
