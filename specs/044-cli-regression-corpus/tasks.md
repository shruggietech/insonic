# Tasks: Maintained CLI regression corpus and audit coverage

**Input**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [coverage contract](contracts/coverage.md)
**Verification**: Tests precede their implementations. Reviewer-owned coverage.md checkboxes are requirements-review prompts, not implementation status.

## Phase 1: Setup

- [x] T001 Verify S044 / codex/044-cli-regression-corpus / specs/044-cli-regression-corpus identity, clean main baseline and #114 scope.
- [x] T002 Complete specify, clarify (zero critical ambiguities), checklist, plan and read-only research in specs/044-cli-regression-corpus.

## Phase 2: Foundational gate

- [x] T003 Run blocking /speckit-analyze across spec.md, plan.md and tasks.md; resolve critical/high inconsistencies before source implementation.

## Phase 3: US1 - Maintained fixtures (MVP)

**Goal/test**: Offline catalog identifies existing fixtures and rejects malformed, stale, corrupt or escaping references.

- [x] T004 [US1] Add failing metadata/path/symbol/duplicate/version/pinned-manifest tests in tests/cli_corpus_test.py, including 64 KiB inventory, 100-fixture/32-scenario and 64-character ID bounds.
- [x] T005 [US1] Add compact version-1 tests/fixtures/cli-corpus.json referencing licensed media and current generators/assertion owners, provenance and expectations (strings at most 512 characters).
- [x] T006 [US1] Implement strict offline inventory validation in scripts/cli_corpus.py; prove deterministic selected PCM/reference expectations through existing helpers in tests/cli_corpus_test.py.

## Phase 4: US2 - Truthful dynamic coverage

**Goal/test**: Current installed help defines every command/option row; observations distinguish assertion levels and reject unsupported semantic success.

- [x] T007 [US2] Add failing contextual-help parsing, accepted-option drift, alias/global/value/literal-separator and secret-value omission regressions in tests/cli_corpus_test.py.
- [x] T008 [US2] Implement derived command/option authority and bounded value-free observations in scripts/cli_corpus.py and expose existing global-option value arity in cmd/insonic/commands.go; enforce at most 20,000 non-help observations and a 1 MiB ledger.
- [x] T009 [US2] Add failing semantic-outcome and async-refusal regressions in tests/cli_help_examples_test.py; remove unasserted succeeded claims in scripts/cli_help_examples.py and retain assertion/scenario context for durable refusals.
- [x] T010 [US2] Integrate observation aggregation and completed-scenario dispositions into scripts/cli_contract_qualification.py without adding another journey; retain source-only companion gaps and parallel helper isolation.
- [x] T011 [US2] Bind nested ledger to CLI hash/platform/version and existing verified archive/inventory identity through scripts/package-desktop.py and affected qualification tests; reject mismatches and unknown identity members.

## Phase 5: US3 - Packaged integration and audit boundaries

**Goal/test**: Existing package journeys use spaces/Unicode destinations, cleanup fails truthfully when owned resources survive, and receipts preserve scope limits.

- [x] T012 [US3] Extend meaningful destination/identity integration tests and rename existing owned install/library roots in scripts/package-desktop.py to include non-Latin text while retaining spaces.
- [x] T013 [US3] Add failing thread-cleanup regressions and assert joined provider/documented/model fixture servers stop in scripts/cli_provider_qualification.py, scripts/documented_workflows.py and scripts/cli_contract_qualification.py.
- [x] T014 [US3] Document catalog/ledger use and explicit verification boundaries in docs/v1.0.1/development.md; record the targeted qualification change in CHANGELOG.md without version bump or public planning links.

## Phase 6: Convergence and verification

- [x] T015 Run /speckit-converge against this task list and append only demonstrated gaps to specs/044-cli-regression-corpus/tasks.md; retain all mandatory integrity/privacy/timing checks.
- [x] T016 Run root npm check/test, complete Python unittest, affected package/site checks and targeted actual native CLI qualification; inspect UTF-8/LF/mojibake/diff and remove owned temporary outputs.

## Phase 7: Authorized publication and bounded review

- [x] T017 Track #114 in the S044 GitHub milestone/Project; commit and automatically push codex/044-cli-regression-corpus, open/attach an ordinary PR with #114 closure and #147 reconciliation scope.
- [ ] T018 Monitor exact-head complete CI to terminal acceptance using the current timing policy; fix failures and preserve canonical GitHub evidence.
- [ ] T019 Address every received Codex finding/thread; allow initial automatic plus one explicit second round, never a third; retain actual reviewed-head identities.
- [ ] T020 Reconcile #147 coverage/defect status with verified S044 results and explicit unverified service/inference/hardware/#154 limits; keep unrelated issue/dependency scopes unchanged.
- [ ] T021 Verify final clean head, checks, threads and Project status; delete owned scratch, leave PR unmerged and hand off for owner final review.

## Dependencies and execution

T001-T003 -> US1 -> US2 -> US3 -> T015-T016 -> T017-T021. US1 catalog validation is the MVP; delivery requires all stories. Read-only command/fixture research ran in parallel. US2 authority/parser tests and independent US3 cleanup tests can be prepared in separate files; shared qualification edits stay sequential. No parallel same-file mutation is planned.

## Implementation strategy

Inventory first, then derive accepted coverage and separate observed/asserted outcomes, then integrate exact existing package identities and owned path/cleanup checks. Preserve current scenarios and add only demonstrated missing coverage. Source and installed-package results retain distinct verification claims.

## Phase 8: Convergence

- [x] T022 Link every command/option observation to its actual catalog scenario, including parallel helpers, and reject unknown scenario IDs per FR-003 and US2/AC1 (partial, HIGH).

## Phase 9: Review and hosted failure corrections

- [x] T023 Replay value-free authority-building observations, aggregate schema-owned failure categories and always check server liveness after cleanup exceptions per FR-003/FR-008 and first Codex review.
- [x] T024 Preserve Windows Unicode extractor filenames through UTF-8 stdin, verify native extraction, and close the package failure diagnostic SQLite connection per FR-006/FR-008 and the initial hosted package failure.
- [x] T025 Reproduce the hosted ExifTool Unicode installation failure, transform only the pinned packaged launcher's process manifest before signing, preserve portable source/receipt identity and independently verify unsigned bytes; share the existing setup PE resource writer and test unchanged resources per FR-006 and plan decision 8.
- [x] T026 Reproduce the later LadybugDB failure under a non-Latin parent with a Legacy process manifest, apply process-local UTF-8 to Windows CLI/desktop outputs before qualification/signing/inventory, preserve resource bytes and require the manifest during independent current-source package verification per FR-006/FR-009.
- [x] T027 Keep npm advisory audits enabled while bounding fetch time/retries inside the existing 180-second CI input stage; preserve the 400-second artifact waits, original readiness clock and all required checks per FR-009 after hosted advisory-service stalls.
