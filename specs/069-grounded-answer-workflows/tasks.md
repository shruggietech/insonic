# Tasks: Grounded answer workflows

**Input**: specs/069-grounded-answer-workflows/{spec,plan,research,data-model}.md and contracts/answers.md.
**Tests**: Required by the approved issues and autopilot. Write meaningful failing regression tests before implementation; preserve security/tenancy checks.

## Phase 1: Setup

- [x] T001 Create exact S069 branch, selector, specification and quality checklists in specs/069-grounded-answer-workflows/.
- [x] T002 Resolve planning research and document contracts, data model and quickstart in specs/069-grounded-answer-workflows/.

## Phase 2: Foundational gate

- [x] T003 Complete requirements review and blocking consistency analysis in specs/069-grounded-answer-workflows/analysis.md; resolve every critical/high finding before implementation.

## Phase 3: US1 - Inspect grounded synthesis (#224)

**Goal**: Bounded provider output carries independently validated current citations and explicit uncertainty.
**Independent test**: Fixture providers and retained evidence verify exact quotes, visible membership, foreign/time-shift rejection, source instructions, limits and finite repair without changing recordings.

- [x] T004 [P] [US1] Add failing strict synthesis, visible-context, quote/paraphrase, cardinality and uncertainty tests in internal/answers/synthesis_test.go; enforce128claims,128distinctreferences and quote text equal to supporting quote.
- [x] T005 [P] [US1] Add failing shared structured transport security tests in internal/assistance/generation_test.go for both adapters, redirects, hosted/local route checks, response limits, malformed output and cancellation.
- [x] T006 [US1] Implement bounded structured transport reuse in internal/assistance/http.go and ollama.go, preserving query Propose behavior; enforce131072byte serialized request and524288byte maximum response.
- [x] T007 [US1] Implement synthesis request/visible-evidence validation and immutable answer types in internal/answers/synthesis.go; allow512character model identity without changing preparation semantics in contracts.go.
- [x] T008 [US1] Implement bounded primary/repair orchestration in internal/app/answer_synthesis.go with max_repairs0or1(default1), one30sdefault/60smax deadline, generated diagnostics/response digests and current-source validation.

## Phase 4: US2 - Run and recover durable work (#225)

**Goal**: Complete unattended CLI/pipeline workflow with durable phases and truthful publication/readback.
**Independent test**: Request/wait/inspect/export/fail/retry/cancel/restart/invalidate against controlled providers and supported backends.

- [x] T009 [P] [US2] Add failing lease/cancel/source-epoch/receipt/retry tests in internal/catalog/answer_work_test.go and internal/app/answer_work_test.go, including unknown publication recovery and unchanged sources.
- [x] T010 [P] [US2] Add failing answer-only/combined pipeline and frozen election tests in internal/pipeline/answer_test.go and internal/app/answer_work_test.go.
- [x] T011 [US2] Implement transaction-capable epoch and atomic terminal acceptance in internal/catalog/answer_evidence.go and answer_work.go, preserving recordWork receipt proof and all backend semantics.
- [x] T012 [US2] Implement frozen admission, linked retries,180000msaggregate work limit, phase checkpoints, candidate reconciliation and1MiB artifact storage/readback in internal/app/answer_work.go and answer_artifact.go; integrate work.go and answers_cache.go.
- [x] T013 [US2] Add answer_synthesis pipeline validation, answer-only inspection/readiness and frozen configuration in internal/pipeline/definition.go, internal/app/configured.go and worker_readiness.go.
- [x] T014 [US2] Add answers.request/retry contracts, CLI help/human views and schema validation in internal/contracts/, cmd/insonic/, internal/runtime/deadline.go and schemas/v1.1.0/; preserve existing prepare/show/export/validate compatibility.
- [x] T015 [US2] Extend real graph/S3/catalog parity and backup/restore regressions in internal/app/answers_graph_test.go, answers_s3_test.go and internal/catalog/answer_work_test.go.

## Phase 5: US3 - Evaluate and recommend release (#226)

**Goal**: Separate measured protocol/provenance from semantic quality and retain a compatibility/readiness packet.
**Independent test**: Known synthetic corpus with duplicates/withholding/contradictions produces exact declared metrics and inspectable failures; optional model mode identifies its election.

- [x] T016 [P] [US3] Add versioned synthetic corpus and failing scoring/report-integrity tests in tests/fixtures/answers/manifest.json and tests/answer_reference_test.py, including zero-denominator null/not-applicable and false-claim/valid-citation cases.
- [x] T017 [US3] Implement strict corpus scoring and explicitly elected installed-CLI evaluation in scripts/answer_reference.py and scripts/qualify-answers.py, keeping expected rubrics out of provider input and retaining model/configuration/corpus identity and failures.
- [x] T018 [US3] Extend installed answer journey, provider fixtures, corpus ownership and executable help in scripts/cli_answer_qualification.py, cli_contract_qualification.py, cli_help_examples.py and tests/fixtures/cli-corpus.json.
- [x] T019 [US3] Add installed Linux PostgreSQL/S3/ArcadeDB answer campaign using maintained CI service readiness in scripts/ and .github/workflows/ci.yml; default Windows/Linux packages retain existing campaign coverage.
- [x] T020 [US3] Update authoritative docs/v1.1.0/answers.md, pipelines.md, cli.md and development guidance, CHANGELOG.md and specs/069-grounded-answer-workflows/release-recommendation.md; recommend1.2.0 while keeping current1.1.0 versions aligned.

## Phase 6: Convergence, qualification and handoff

- [ ] T021 Run speckit-converge, affected local checks, npm run check, npm test and site/package parity; retain results in specs/069-grounded-answer-workflows/verification.md and .local/s069/ without private content.
- [x] T022 Commit/push and open the authorized official PR with closing references to #224-#226 (and #210 only if fully met); record the PR in specs/069-grounded-answer-workflows/verification.md or its retained PR handoff receipt. No new tracking issue.
- [ ] T023 Resolve every external finding within at most two review rounds, retain exact-final-head CI terminal audits and artifact digests, verify project/issue/review state and ping owner for final review/merge in the retained PR handoff receipt.

## Dependencies and execution order

T001-T003 precede all implementation. T004/T005 are independent failing-test groups; T006-T008 deliverUS1. T009/T010 can be prepared independently, then T011-T015 deliverUS2 usingUS1. T016 corpus/rubric work can be independently tested; T017-T020 integrateUS1/US2. T021-T023 require all story acceptance. Chronology follows these phases.

## Parallel examples

US1: pure answer validation tests and transport tests use separate files. US2: catalog publication-fence tests and pipeline definition tests use separate files. US3: corpus scoring tests can be developed independently of installed fixture wiring. These are dependency opportunities, not blanket permission to spawn implementation agents.

## Implementation strategy

Prove bounded synthesis first, then durable publication/recovery, then complete installed workflows and evaluation. Reuse maintained services rather than parallel infrastructure. Every story is independently testable, but the authorized slice includes all three. Qualification and external review receipts complete handoff; owner controls merge.

## Phase 7: Convergence

- [x] T024 CRITICAL: Bind every installed answer invocation and held fixture wait to one original enclosing campaign deadline in scripts/cli_answer_synthesis_qualification.py and scripts/qualify-answers.py per FR-016 and Constitution V (partial).
