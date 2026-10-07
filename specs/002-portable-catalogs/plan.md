# Implementation Plan: Portable catalogs and durable jobs

**2026-10-07 contract supersession**: S005/#21 replaces append-only computed transcript/assignment histories and frozen copied dataset evidence with one current embedded Cueson document, external known-speaker mappings and current-reference or invalidated downstream records. Original sources and noncontent model/run lineage remain retained. Historical migration definitions and the recorded S002 verification evidence remain unchanged.

**Branch**: `codex/s002-portable-catalogs` | **Date**: 2026-10-05 | **Spec**: [spec.md](spec.md)

## Summary

Implement a shared SQL catalog using the existing SQLite and pgx drivers, portable DDL, typed relational records and transactional workspace revision/receipt authority. Integrate durable timer qualification jobs into app/IPC, add history/catalog transfer commands and gate expensive qualification by affected paths.

## Technical Context

**Language/Version**: Go 1.27.1, existing Node/Python tooling.

**Primary Dependencies**: Existing go-sqlite3 1.14.52 and pgx 5.11.0. Qualified Wails remains 2.14.0; no dependency upgrade.

**Storage**: SQLite with per-connection FKs/WAL/full sync/immediate transactions; PostgreSQL 18.6 fixture with short workspace-row locks and server-time leases.

**Testing**: Shared Go catalog contracts, app/restart/IPC tests, PostgreSQL integration and cross-backend transfers, existing npm/Python/site checks.

**Target Platform**: Windows, Linux and macOS, existing architecture matrix.

**Project Type**: CLI-first shared runtime with desktop bridge.

**Performance Goals**: Recovery within ten seconds after expiry; bounded transactions outside work; CI jobs below ten minutes.

**Constraints**: Signed int64 revisions/generations/nanoseconds, no resolved credentials in records/errors, tenant FKs, append-only evidence, stable checks. Restore only empty authority.

**Scale/Scope**: Core settings/profiles/metadata/associations and concrete speaker corpus/training/model schema reservations. No media pipeline or graph execution.

## Constitution Check

Pre-research and post-design: PASS. Shared CLI/runtime authority, both catalogs, exact Cueson values, nonsecret profiles, no human approval gate, UTF-8/LF and truthful docs remain enforced. Owner authorizes push/PR and retains final merge. Security/tenancy tests are mandatory.

## Project Structure

```text
internal/catalog/       # typed records, migrations, revisions/jobs/outbox/snapshots
internal/app/           # durable timer work and recovery lifecycle
internal/runtime/       # catalog startup errors
internal/contracts/     # aligned request/error contracts
cmd/insonic/            # history, catalog show/export/restore/migrate
schemas/v0.0.0/         # runtime operation schema
scripts/ci-scope.mjs    # conservative component selection
scripts/qualify.py     # manifest pin drift guard
.github/workflows/ci.yml
specs/002-portable-catalogs/
```

## Decisions

1. Serialize mutations per workspace in short transactions. PostgreSQL locks the workspace row; SQLite obtains writer ownership before reads. Unrelated PostgreSQL workspaces stay independent.
2. Typed domain structs and portable tables use scoped FKs. Evidence is insert-only; settings have CAS/history. Structured reports/options use lossless JSON without backend JSON dependence.
3. Admit operation digest/receipt atomically. Retry only recognized bounded serialization/deadlock/busy failures; reconcile unknown commit results through the same operation ID.
4. Sample server time after locking. Renewable leases fence completion; recovery claims only expired/interrupted work. Shutdown records interruption, cancellation remains terminal until explicit retry.
5. Ordered target checkpoint/claim exposes only checkpoint+1; exact event identity and generation fence acknowledgement. Graph execution stays deferred.
6. Export one consistent transaction with version/digest. Restore atomically into empty authority, expire imported ownership and record recovery.
7. Explicit PostgreSQL routing uses injected/session opaque credentials (username/password JSON in memory), verified TLS, no ambient pgpass/routing or fallback. Native/encrypted persistence stays issue #5. Separate migrate from runtime opening for least privilege.
8. Freeze pins. New PR #17 is independently green; #18 changes Wails without qualification metadata. No separate dependency PR is merged in this owner-merge-bounded slice. Add pin guard.
9. Preserve named CI jobs, conservative unknown-file fallback and relevant platform/backend tests; gate expensive Cueson/webview steps by actual input paths.

## Delivery sequence

Specify/clarify -> research/design -> tasks/analyze -> TDD implement -> CI parity -> converge -> automatic publication -> automatic review and at most one second round -> owner merge handoff.
