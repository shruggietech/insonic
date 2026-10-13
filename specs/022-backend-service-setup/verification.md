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

## Remaining external acceptance

Exact-head CI, live alternative-backend qualification and up to two review rounds follow PR publication. No merge/tag/release is authorized by this slice. Real-model inference and attended native GUI behavior are not claimed by deterministic protocol/rendered tests.

## 2026-10-13: First external review and CI correction

Initial head 888c541 received three code review findings. Regression tests reproduce premature graph publication, lost pending identity after backend mutation and lost version/reachability evidence after metadata failure. The selected graph now shares runtime initialization, with schema/owner/version checks before publication; pending intent binds the directory before mutations; partial metadata reports retain observed running/reachable state. Explicit test-only factories isolate untagged unit tests; the tagged native test exercises real Ladybug initialization.

Initial Linux desktop CI caught the omitted cancellation ref initializer. Fixed the ref and added the configured frontend TypeScript check to local verification; type checking and rendered tests pass.

Post-remediation full Go tests/vet, repository checks/tests, configured frontend type check/rendered tests and site build pass. Tagged native qualification now includes backendsetup so the real selected Ladybug graph bootstrap test runs in CI. Initial run 38371824332 passed live adapter fixtures, including credential-before-PostgreSQL and non-mutating catalog preflight. Final-head qualification follows the correction push.
