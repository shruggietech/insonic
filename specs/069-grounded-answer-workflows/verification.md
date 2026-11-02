# Verification record

## 2026-11-02: Implementation and local qualification

Scope is S069, existing issues #224–#226, branch `codex/069-grounded-answer-workflows`, feature directory `specs/069-grounded-answer-workflows`. No new tracking issue, release publication or merge is authorized by this slice.

- Spec Kit specify, clarify, checklist, plan, tasks, analyze and implement phases completed; the initial analysis mapped all 22 requirements/success criteria to 23 tasks with no blocking findings.
- Focused failing-first regressions cover provider bounds, forged/current references, immutable candidate recovery, atomic cancellation/epoch/lease acceptance, linked retry, frozen pipeline settings and typed schemas.
- Full ordinary `go test ./...` passed. Local logs are retained under `.local/s069/`.
- Schema checks pass with 31 schemas, 28 contracts and 36 examples, all at version 1.1.0.
- The actual source-built Windows CLI passes all seven synthetic evaluation cases and request/wait/show/export/fail/retry/cancel/partial/stale-source lifecycle checks. Receipt: `.local/s069/answer-cli-receipt.json`.
- Five conservative scoring/corpus regressions pass. A valid citation attached to a false claim fails the support rubric; fixture mode cannot label itself real-model quality.

An initial local native CLI build used the wrong build tag and could not publish its graph. Rebuilding with the maintained `system_ladybug` tag corrected the harness. A provider-busy assertion was corrected to the established retryable exit code 4. These failed diagnostics remain retained and are not acceptance evidence.

## 2026-11-02: Backend, documentation and convergence checks

- The actual source-built Windows CLI completes the same seven-case/lifecycle campaign on PostgreSQL, S3 and ArcadeDB with one owned runtime and natural cleanup. Receipt: .local/s069/answer-backends.json. Required CI runs this maintained campaign on Linux as well.
- Native/integration answer regressions pass on SQLite/PostgreSQL, filesystem/S3 and LadybugDB/ArcadeDB, including terminal receipt restoration. Restart after byte publication but before work-reference persistence is reconciled without another provider call.
- Generic work retry returns the sanitized answer work view; a regression verifies that frozen endpoint/payload details are absent.
- npm run check passes; npm test passes all 223 tests. Go vet passes. Site tests pass all 18 tests, and the production documentation export checks 200 pages and their links/anchors.
- The Python suite executed 749 tests with five skips and one setup-class error caused by an ambient compiler mismatch. All seven affected Windows setup tests pass with the repository-pinned Go compiler. Final CI repeats the complete suite under its pinned compiler.
- The first convergence assessment found one partial timing requirement: nested answer fixture calls/waits lacked a common original deadline. T024 binds them to the earlier caller/campaign deadline. Both actual CLI campaigns pass after that correction.
- The follow-up convergence assessment checked 17 functional requirements, five success criteria, ten acceptance scenarios, eight plan decisions and five constitutional principles. No buildable gaps remain; tasks.md was unchanged by the clean convergence assessment. Operational CI/review/handoff gates remain pending.

## 2026-11-02: First external review and CI findings

Official PR #247 is open at https://github.com/shruggietech/insonic/pull/247. No tracking issue was created. The first automatic review found two valid P2 recovery problems. Failing-first regressions reproduced missing-publication restart failure and truncated HTTP/Ollama bodies misclassified as malformed model output. The revised checkpoint retains the exact bounded bytes before publication, resumes confirmed missing/pending publication without inference, rejects corrupt retained bytes and omits those bytes from public progress and accepted results. Interrupted response reads remain typed transport failures and fail durable work after one attempt. Focused answer and transport regressions pass.

Initial PR CI run 41459447714 passed core, adapter, foundation and documentation jobs. Four package workers exposed an expected CI inference refusal missing from the emitted-help fixture outcome map. The fixture now records that specific refusal and serves a bounded deterministic answer outside CI; the separate installed answer campaign continues to prove successful workflows. The complete Go suite, npm check, all 223 JavaScript tests, seven help-example helper tests and the rebuilt actual CLI seven-case/lifecycle campaign pass after the corrections. The failed terminal audit and artifact inventory were readback-verified outside Actions in https://github.com/shruggietech/insonic/pull/247#issuecomment-6579747473 before the next full run. Final package acceptance remains pending.

## Outstanding acceptance

Complete external reviews (maximum two rounds), exact-final-head CI/package evidence and retained terminal audit before owner handoff. Real model evaluation remains not elected. Final accepted CI, artifact and review identities belong in the retained PR handoff receipt.
