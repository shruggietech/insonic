# CI acceptance state and timing

Preparation is pending, ready or failed. Ready requires successful probe and successful verified warm transfer with skipped cold final stage, or successful cold final stage. Failed/cancelled/ambiguous outcomes cannot authorize consumers.

Acceptance input contains exact repository, revision, run, attempt, boolean election flags, top-level results and readiness timestamp. Elected results must be success; non-elected results must be skipped. Missing and unknown states reject.

Timing uses current-attempt run_started_at, the readiness boundary, evaluation time and, for terminal audit, the latest completed_at across the complete current-attempt job inventory. All timestamps are finite, ordered and identity-bound. Preparation = readiness - attempt start; qualification = endpoint - readiness; combined = endpoint - attempt start. Initial request queue (created_at to run_started_at for attempt one) is reported separately. Queue time after attempt start belongs to its phase. Documentation-only preparation ends at the boundary too; its scope/setup overhead remains visible.

The provisional aggregate records evaluation time, 660-second limit and 15-second finalization reserve. Only a completed attempt supplies final elapsed compliance. Receipts do not contain access tokens.

## Prepared Windows Go dependencies

An exact-source/run/attempt manifest binds Go/module/recipe hashes, resolved compiler and MSYS2 packages, pinned native assets and six existing package contexts, including CGO-disabled trimpath setup dependencies. Each context records only standard or explicitly non-main-module library targets. A bounded archive digest binds an isolated Go build cache and module cache. Consumers validate the complete identity and all archive paths before activating fresh cache directories. This is a dependency optimization; consumer product compilation and acceptance remain required after readiness.
