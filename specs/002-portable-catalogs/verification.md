# Verification: Portable catalogs and durable jobs

## Local evidence

- Repository integrity: `npm run check` passed (207 project text files, 15 schemas, 13 contracts, 14 examples); UTF-8/LF, links, versions and governed kit bytes passed.
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

Hosted Linux/macOS/Windows qualification and all PR checks must pass on the final head. Every reported review finding must be addressed and its thread resolved. At most one explicit second Codex review may be requested after the automatic first round. The owner retains the final squash merge and branch deletion decision.

Artifact-store operations, persistent encrypted/native credentials, media processing, graph consumers, model production and installable release packages remain separate slices. This slice supplies their typed durable catalog relationships and handoff contracts.
