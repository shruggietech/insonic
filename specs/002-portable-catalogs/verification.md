# Verification: Portable catalogs and durable jobs

## Local evidence

- Repository integrity: `npm run check` passed (210 project text files, 15 schemas, 13 contracts, 14 examples); UTF-8/LF, links, versions and governed kit bytes passed.
- Maintainer tests: `npm test` passed all 46 tests; Python unittest discovery passed all seven tests.
- Core acceptance: `go test ./...` and `go vet ./...` passed.
- Both catalog backends: shared acceptance, concurrency, exact timestamps, scoped provenance, frozen corpus/training/model associations, cancellation, ordered outbox and SQLite/PostgreSQL round trips passed against the pinned PostgreSQL 18.6 fixture.
- Race detection: `go test -race -tags integration ./internal/catalog ./internal/app ./internal/runtime` passed after final convergence changes.
- Convergence: configured engine constraints, populated snapshot schema, 128-attempt pagination, cancellation/publication races, credential/TLS failures, ambient-route isolation and recomputed snapshot semantic corruption passed.
- Regression evidence: tests first reproduced recovery overwriting shutdown evidence, cancellation replay detaching a retry, and capacity blocking acceptance reconciliation. Their fixes passed under race detection.
- Documentation: the site generated 27 static routes; 25 offline documentation pages passed relative asset/link/anchor checks.
- Windows qualification: pinned native SQLite/Ladybug operations, Cueson checksums/round trip, cross-process CLI owner reuse and durable history export, native secret-service availability, packaged help and desktop bridge/native webview passed. Receipts are ignored build outputs and hosted CI artifacts.

## Analysis and convergence

All 13 functional requirements and five success criteria map to acceptance work. Analysis found no unresolved constitution conflicts or product questions. Convergence tasks T017-T019 closed observed implementation gaps; no mandatory attended review or backend fallback was added.

## Publication gates

The first published head `3007e51` passed all six hosted checks in [run 37397308255](https://github.com/shruggietech/insonic/actions/runs/37397308255): foundation (10 seconds), docs (30 seconds), adapter fixtures (1 minute 43 seconds), macOS (3 minutes 10 seconds), Linux (3 minutes 46 seconds) and Windows (5 minutes 56 seconds).

The first Codex round reported credential-key alias admission and acknowledgement reconciliation after expiry. Shared-backend regressions reproduced both failures. Credential-option validation now normalizes case/separators and covers common credential aliases while retaining opaque references and ordinary tokenizer/token-limit options. Acknowledgement retries validate immutable event identity and a durable acceptance receipt before reporting an already committed outcome; new writes still require current ownership. Both fixes passed race-enabled catalog/app/runtime acceptance on SQLite and PostgreSQL.

The second and final requested round reviewed `d14af89`, which also passed all six checks in [run 37397982470](https://github.com/shruggietech/insonic/actions/runs/37397982470). It reported asymmetric snapshot file-size limits and missing acknowledgement evidence for restored checkpoints. File input now validates through a disposable spool without a fixed size cap; an actual catalog export exceeding 64 MiB restores intact. Checkpoints require the deterministic acknowledgement receipt, matching intent/result and ordered accepted revision. Shared-backend tests also reject checkpoint rewinds and scalar type coercion. Application shutdown cancels blocked catalog calls before waiting for its mutex.

The final head must pass all configured checks. Every reported review finding must be addressed and its thread resolved. Exactly one explicit second round was requested; no third round is authorized or requested. The owner retains the final squash merge and branch deletion decision. GitHub PR #19 provides the final hosted check and review record.

Artifact-store operations, persistent encrypted/native credentials, media processing, graph consumers, model production and installable release packages remain separate slices. This slice supplies their typed durable catalog relationships and handoff contracts.
