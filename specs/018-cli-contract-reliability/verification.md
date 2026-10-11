# S018 verification

## Local checkpoint

- Spec Kit specify, clarify, checklists, plan, tasks and analyze completed for the nine approved issues. Converge compared the implementation with every FR/SC; no remaining implementation gap was identified. Hosted platform/backend and review gates remain pending until publication.
- npm run check and npm test: passed (83 tests).
- go test ./... and go vet ./...: passed. Additional date correction, item override/replay, acquisition deadline/checksum and historical snapshot/backup regressions passed in the affected packages.
- Python maintainer suite: passed (141 tests).
- Frontend type checking and tests: passed (73 tests).
- Site and offline documentation build: passed (78 pages), including the current CLI reference.
- Installed generic CLI matrix: passed 143 command/alias help paths in all three spellings inside/outside workspaces, zero created help files, argument errors, settings mutations/reset, first speaker-model cursor, protected vault deletion, conflict and checksum dependency causes.
- Historical fixtures retain workspace/profile identities, original source bytes, generic failure meaning and journal receipts through snapshot/backup restore. Current imports of old tools files retain executable pins and do not rewrite their source.
- UTF-8, BOM, line-ending, mojibake, schema/navigation/link/brand checks and git diff --check: passed. Historical v1.0.0 documentation/schema directories and upstream Cueson/brand bytes remain unchanged.

## Remaining delivery gates

Native CLI qualification passed, including exact nanoseconds and typed causes after owner restart. Extracted Windows CLI/desktop packages are the remaining local delivery gate. Linux/macOS/Windows native packages, alternative backend fixtures and exact-head CI will be verified on the official PR. Initial external reviews and at most one requested second round must be satisfied before the owner handoff. No acoustic inference, merge, tag or release is part of this slice.

## Initial review correction

The initial external Codex review identified a historical import-manifest/current IPC version mismatch. A failing CLI parser-to-shared-schema regression reproduced it. Admission now normalizes the operation copy while preserving source bytes and exact timestamps. The adjacent model-manifest boundary retains historical declared versions and content digests through explicitly packaged schema references; its register/acquire regression also passes. Schema help, offline compilation and release/catalog test fixtures include those historical dependencies.

All Go packages, npm checks and 83 tests, the 141-test Python suite (plus the 35-test affected release suite), and site/offline documentation pass after this correction. Initial Linux/macOS native and both package variants plus alternative-backend fixtures pass on the PR. Windows hosted stages remain pending. Local extracted-package validation rejects a monolithic media source cache without the required split dependency receipts; the owned staged build is regenerating that provenance. Final-head package/CI evidence and the second external review round are recorded on PR #71 before owner handoff.
