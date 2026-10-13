# S022 verification

## 2026-10-13: Local acceptance

- Spec Kit specify/clarify/checklist/plan/tasks/analyze/implement and convergence executed. Six backend choices and native Ollama routing remain in scope. Extension hooks absent.
- Initial backend/protocol tests failed for missing implementations, then passed after implementation. Field corrections exclude input values, protected credential bootstrap precedes catalog opening, and failed/repeated provisioning preserves committed selections.
- npm run check: 24 schemas, 22 contracts, 29 examples; repository text/encoding/link/brand integrity passed.
- npm test: 84 tests passed after registering the new contract/example and reusing shared backend definitions rather than duplicating their reference documentation.
- go test ./... and go vet ./... passed. The new integration package compiles locally; live PostgreSQL fixture checks await adapter CI.
- Python maintainer suite: 155 tests passed, including Windows credential lifecycle.
- Rendered frontend suite: 75 tests passed, including all six backend forms, protected initial values and explicit hosted Ollama model election.
- Site production/offline build: 78 documentation pages checked.
- Convergence corrected native request UUID generation for opaque origins and shared backend schema references. No unresolved product-scope finding remains.
- Frozen docs/schemas v1.0.0 remain byte-unchanged. Windows checks used verified hidden redirected launch paths.

## Verification boundary

Qualification includes exact-head CI, live alternative backends and two completed Codex review rounds. No merge/tag/release is authorized by this slice. Real-model inference and attended native GUI behavior are not claimed by deterministic protocol/rendered tests.

## 2026-10-13: First external review and CI correction

Initial head 888c541 received three code review findings. Regression tests reproduce premature graph publication, lost pending identity after backend mutation and lost version/reachability evidence after metadata failure. The selected graph now shares runtime initialization, with schema/owner/version checks before publication; pending intent binds the directory before mutations; partial metadata reports retain observed running/reachable state. Explicit test-only factories isolate untagged unit tests; the tagged native test exercises real Ladybug initialization.

Initial Linux desktop CI caught the omitted cancellation ref initializer. Fixed the ref and added the configured frontend TypeScript check to local verification; type checking and rendered tests pass.

Post-remediation full Go tests/vet, repository checks/tests, configured frontend type check/rendered tests and site build pass. Tagged native qualification now includes backendsetup so the real selected Ladybug graph bootstrap test runs in CI. Initial run 38371824332 passed live adapter fixtures, including credential-before-PostgreSQL and non-mutating catalog preflight. Final-head qualification follows the correction push.

## 2026-10-13: Second external review remediation

Head c1ad24b passed the complete Windows/Linux CI run 38374317969, including live adapters, native desktop/engine, CLI/desktop packages and candidate integrity. The second and final requested Codex review completed on that head with three findings. Regression tests reproduced omitted S3 access checks, incorrect completed-retry credential mode admission and suppressed selected-model metadata errors before implementation corrections.

Provisioning now requires non-mutating selected S3 bucket access before publishing configuration, while leaving object writes untested. Completed retries hold canonical workspace ownership, reread configuration and verify both retained plan/mode identity and persisted credential selection. Failed model details expose redacted per-model corrections; a selected-model failure returns partial metadata error so discovery retains observed running/reachable evidence and reports metadata-failed. Rendered discovery exposes both candidate and per-model corrections without adoption.

Targeted shared Go bootstrap/assistance/app/CLI/desktop tests and full go test ./... / go vet ./... pass after remediation. npm run check and all 84 repository tests pass; frontend TypeScript check and all 76 rendered tests pass; production/offline site build and 78-page link checks pass. New-head CI qualification follows. No third review request is permitted or will be posted.

Corrected the new S3 cancellation fixture after Linux CI run 38376803237 showed its 100 ms timer could expire during catalog initialization. Cancellation is now signaled by receipt of the HEAD request, with an outer deadline as a hang bound. This verifies in-flight bucket cancellation without relying on initialization speed; production behavior is unchanged. Post-remediation Python qualification also passed all 155 tests, including native Windows credential lifecycle and cross-process restart.

## 2026-10-13: Qualification and bounded-review closure

Implementation/test head `22a8f16ae838e17a340426c1746d99d4115bb29d` passed [CI run 38377279095](https://github.com/shruggietech/insonic/actions/runs/38377279095): repository/docs, both platform core suites, live PostgreSQL/S3/ArcadeDB fixtures, native engine/desktop, all four Windows/Linux CLI/desktop packages and candidate integrity. Source-build cache lanes skipped only after verified cache reuse. The new event-driven cancellation regression passes on Linux as well as repeated local runs.

[Official PR #76](https://github.com/shruggietech/insonic/pull/76) contains the final branch/check evidence. The automatic first review completed on `888c541`, and the single requested second/final review completed on `c1ad24b`. GitHub labels both as Code Review (the second request asked for security review). Three findings in each round were corrected with regression coverage, answered and resolved. Direct review-thread inspection confirmed all six resolved and no additional page. No third review request was posted.

Final Spec Kit convergence inspected 16 functional requirements, 6 success criteria, 12 story acceptance scenarios, 7 plan design decisions and all 5 constitution principles against the named code scope. It found one MEDIUM documentation mismatch (default graph location), appended T029 and corrected it. Repository text checks and the production/offline 78-page export pass after that correction. No remaining buildable gap or product-scope deviation is identified. Tasks T001-T029 are complete.

The final documentation/evidence reconciliation commit changes no product code; its exact PR head must also pass CI before owner handoff. Final-head check evidence is retained on PR #76 rather than creating another evidence-only commit after that qualification. Both linked issues remain In review until the owner squash-merges and deletes the branch. Software, docs and schemas remain the unreleased 1.0.1 candidate; frozen 1.0.0 artifacts are unchanged.
