# S003 verification

## Scope and implementation

2026-10-06: Actual session S003, branch codex/s003-artifact-storage and feature directory specs/003-artifact-storage implement issue #4. That issue retains original roadmap label S004. Dependencies remain pinned. All three stories are implemented: verified filesystem/S3 publication and recovery, ranged reads and leased materialization, and catalog-fenced retirement. Catalog schema 2 upgrades schema 1 and accepts verified legacy snapshots without manufacturing lifecycle evidence.

## Spec Kit gates

Specify, clarify, requirements/storage checklists, plan, tasks and analyze completed before implementation. No unresolved product clarification or constitution exception. Test-first authority and adapter/service work exercised failing baselines before implementation.

Convergence found HIGH FR-003 recovery claim renewal and HIGH FR-009 latest-receipt binding gaps, plus MEDIUM FR-008 response bounds and FR-010 process-level CLI evidence gaps. T020-T023 close them. Final assessment finds no remaining buildable requirement gap. Complete journals stay in snapshots; runtime receipts expose bounded counts.

A second assessment found HIGH FR-003 abort-after-unknown-completion could abandon final bytes. T024 makes both adapters refuse that abort, preserving pending reconciliation. The shared S3/filesystem matrix and lost-response service test passed, followed by final artifact race checks. Final convergence is clean.

## Local automated evidence

- npm run check: passed, UTF-8/LF text, links, governed brand bytes and shared release version; 15 schemas, 13 contracts and 14 examples.
- npm test: 46 passed.
- Documentation/site build: passed, 27 routes, 25 documentation pages and offline help with local links/anchors validated.
- go test ./... and go vet ./...: passed.
- Python qualification tests: seven passed.
- go test -race ./internal/artifact ./internal/catalog ./internal/app: passed after convergence fixes.
- Pinned PostgreSQL 18.6 and S3Mock 5.2.3 integration: passed. Both catalog backends execute shared fencing, retain/retire race, restore validation and schema-upgrade tests. S3 covers empty/binary/multipart bytes, ranged reads, resume without re-uploading confirmed part one, abort, full materialization/lease/retirement journeys and exact deletion.
- Deterministic transports prove explicit credential signing, redacted missing credentials and refusal of ignored range requests.
- Failure injection proves lost completion reconciliation, stale generation refusal, renewed slow materialization/recovery and abort waiting for an active publisher to stop.
- Built CLI qualification: passed owner reuse, durable cancellation/retry/history, catalog export/restore and artifact publication replay, verification, materialization, lease renewal/release, retention and cache cleanup.
- Windows pinned Cueson/Ladybug/SQLite native qualification: passed.
- Windows desktop bridge and hidden native webview qualification: passed with rebuilt offline help.
- Windows native secret provider: available; credential values not returned.

## Publication gates

Official PR #20 published. Two review rounds are complete; no further review requests are permitted. Owner retains final merge authority. Final security fixes and their CI are pending.

2026-10-06: Published PR #20 at 2d65f00. Initial hosted macOS and fixture runs exposed overly short test-only leases (10-90 milliseconds), racing SQLite I/O and runner scheduling. Cache expiry now shortens an already materialized normal lease, and slow-work tests use three-second authority with reads extending beyond that duration. The expiry, renewal and stale-authority assertions remain intact. Initial external code review is running; no second request has been made.

2026-10-06: 328aefd passed all six hosted checks in run 37510225600 (longest job 4m15s). First external code review completed on 2d65f00 with four findings. Corrections namespace keys by workspace/profile revision, flush linked metadata before admission/recovery, checkpoint S3 completion intent and keep indeterminate aborts pending, and atomically mark typed locations retired. Regression tests cover shared-root/bucket isolation, injected flush failure, delayed/missing completion responses and retired-location snapshot parity on both catalogs. Physical power-loss testing is not claimed. The following code/security request is the second and final allowed round.

2026-10-06: 1b213d1 passed all six hosted checks in run 37511146495 (longest job 4m59s). All four first-round threads were replied to and resolved. The second round used a combined code/security comment, then a security-only comment within that same round because the combined request started only Code Review. The bot presents both commands under its Code Review summary. No additional round will be requested.

2026-10-06: Second-round review completed with source symlink-swap and remote plaintext transport findings. Source staging now atomically opens without following the final symlink/reparse point, validates regular status through that handle and copies from it; Unix NONBLOCK also prevents FIFO substitution from hanging validation. Tests substitute an opened source path and a symlink before opening. Windows symlink creation may be unavailable on a test host; that scenario explicitly skips where the OS denies fixture creation. S3 rejects remote HTTP before resolving credentials while explicitly local loopback/private literal endpoints retain HTTP support. These enforce existing source and TLS contracts; no product eligibility or approval gate is introduced.

## Demonstrated limits

The pinned S3-compatible fixture and deterministic transport tests qualify the implemented contract; other real provider deployments have not been individually exercised. Hosted Linux/macOS and the unchanged ArcadeDB fixture await PR CI. Native secret persistence, media producers, model registry, full byte backups and installable release packages remain subsequent issues. No per-item review requirement or release claim is added.
