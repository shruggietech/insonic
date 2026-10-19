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

## Phase 8: Convergence

- [x] T027 Retry bounded incomplete GitHub inventory snapshots without accepting them, retain hard failure for malformed/unbounded/ambiguous authority and known failed producers, add race/deadline regressions for all Windows readiness roles, and reverify exact-head CI (FR-018; SC-005, partial).

## Phase 9: Convergence

- [x] T028 Keep runtime-backed catalog inspection in the live example library, require exclusive-maintenance examples never elect a runtime owner, reproduce the isolation violation with the actual installed help corpus before fixing routing, and reverify all 168 examples plus exact-head CI (FR-004/016/018; SC-001/003/005, partial).

## Phase 10: Owner-approved native startup repair

- [x] T029 Update S039 spec/plan/research/contracts and twelve-issue GitHub mapping for the owner-approved #154 scope, then analyze new requirements before native edits (FR-019..022; SC-006).
- [x] T030 Recover controlled owned-process/profile lifecycle evidence for the controller-busy and DOM-readiness signatures using existing bounded diagnostic tools; document the demonstrated failing boundary and remaining uncertainty (FR-019/020).
- [x] T031 Add a failing regression or controlled before/after reproducer and implement the proportional correction to the implicated native/application/process boundary, preserving existing timeout and success rules (FR-020/021).
- [ ] T032 Qualify repeated actual Windows cold/warm menu, close and Alt+F4 routes with/without workspaces, owned browser cleanup, playback release and durable-runtime/work survival, retaining all exact-byte scopes (FR-021; SC-006).
- [ ] T033 Run all required affected checks, reconverge twelve-issue scope, push updated official PR and satisfy exact-head CI <=660 seconds; audit existing resolved reviews without a third round, retain unmerged/auto-merge-off state and record the post-merge-main limit (FR-022; SC-005/006).

## Phase 11: Native convergence

- [ ] T034 Replace the demonstrated unreliable posted menu-selection messages with actual owned-foreground Alt+F then X input, retain active-menu ownership, partial-input refusal, callback/action diagnostics and existing watchdogs; require the shared browser-release proof in source and all package routes, then repeat corrected native and full exact-head CI (FR-019..022; SC-005/006).

- [ ] T035 Restore the specified native Alt+F menu route while the WebView owns focus, using only this application UI thread and owned native window; preserve real SendInput qualification, ordinary text/other shortcuts and callback/normal-exit proofs, then repeat all affected checks and exact-head CI (FR-020..022; SC-006).

## Phase 12: Final-check fixture convergence

- [ ] T036 Keep the general available-artifact setup lease independent of one-second runner timing, retain explicit publication-expiry tests and product defaults, repeat retirement/admission race acceptance, and reverify complete exact-head CI (FR-022; SC-005).

## Phase 13: Owned descendant convergence

- [ ] T037 Identify the owned descendant still reported after normal Alt+F4 host exit with bounded process image class and actual kernel exit state; correct the demonstrated lifecycle/observation cause while preserving existing deadlines, failed cleanup refusal, actual native actions and all exact-head proof gates (FR-019..022; SC-006).
