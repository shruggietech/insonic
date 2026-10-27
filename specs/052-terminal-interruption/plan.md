# Implementation Plan: S052 Terminal progress and interruption acceptance

**Branch**: `codex/052-terminal-interruption` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

**Tracking**: #182 under #147, milestone 36, Project 7, Slice S052. Push/official PR and CI repairs authorized; at most two Codex rounds; stop before merge.

## Summary

Add real hidden terminal transport and a barrier-controlled durable model acquisition campaign to existing CLI qualification. Preserve runtime ownership outside the terminal. Capture real Ctrl+C, natural client exit, continued work, explicit cancellation, timeout/failure and output-mode controls. Require compact exact-executable receipts in source/package acceptance. Fix product code only if this demonstrates a contract failure.

## Technical Context

- Python 3.12 stdlib maintainer tooling; existing Go 1.27.1 CLI/runtime and Node repository/site checks.
- Windows ConPTY via ctypes and existing ProcessTree Job Object ownership; Linux openpty with a fresh interpreter bootstrap establishing a controlling terminal before exec. No new dependency or public interface.
- Existing model acquire, work show/wait/cancel and private runtime contracts. SQLite and owned loopback byte fixtures, no operator data or external services.
- Tests: transport lifecycle/overflow and receipt mutations first; actual host campaign, source/package Windows/Linux CI, full Python/root checks and affected site checks.
- Shared original enclosing deadline plus qualification policy. Target1800s/hard3600s after readiness; provisional60s reserve and separate phase reporting. Ordinary product clocks unchanged.

## Constitution Check

Before research and after design, all five principles pass: approved coverage scope, actual shared CLI, immutable work identities/results, unchanged provider/credential policy and exact bounded evidence. No exceptions or product eligibility restrictions.

## Phase 0 Research and Decisions (2026-10-27)

1. Use ConPTY and Linux controlling PTY rather than pipes or synthetic terminal flags. Spec Kit research agents independently checked terminal ownership and existing fixture integration. Keep Windows console children hidden; no foreground console or native keyboard input.
2. Write Ctrl+C byte to the terminal after a correlated running/count observation. No OS-wide console event, process kill or callback can substitute for terminal delivery.
3. Reuse ProcessTree ownership. Windows client starts suspended, joins its Job Object and resumes with ConPTY. Linux uses a fresh Python bootstrap, not fork/preexec in the multithreaded qualification parent. Drain output concurrently through teardown; bound retained bytes and reject overflow or unknown transport controls.
4. Own an explicit runtime outside the terminal and validate its correlated ready report before client operations. Two-file model acquisition holds the second HTTP response after one verified file. Target the acquisition dependency, not the models.ensure parent. Unique fixture bytes avoid cache reuse.
5. Human success and Ctrl+C require observed running/count progress. JSON and quiet clients use the same held work as the human success client, with final suppression assertions; redirected controls retain existing coverage. Explicit cancellation, short timeout and checksum failure remain distinct.
6. Require natural client exits before closing terminals. Failed/forced teardown cannot pass. Release all barriers, join HTTP handlers, let the owned runtime reach its ordinary idle exit, verify channels/processes and remove only generated fixture files. Cleanup is evidence, not a fallback acceptance path.
7. Add a terminal scenario to the maintained CLI catalog/ledger and a strict nested receipt. Bind executable SHA256; parent retains archive/inventory/source identity. Validate terminal receipts at release qualification acceptance as well as production. No new public contract directory is needed.
8. Keep test protocol failures/mutations and exact-source CI evidence distinct from real inference and installed-release behavior. Append concise docs/changelog/internal records; retain compact audited receipts before reruns and remove owned scratch after readback.

## Project Structure

- scripts/terminal_transport.py: bounded platform terminal lifetime and actual interrupt.
- scripts/cli_terminal_qualification.py: durable work campaign, strict receipt validation and cleanup.
- scripts/cli_contract_qualification.py and cli_corpus.py plus tests/fixtures/cli-corpus.json: shared source/package integration and coverage accounting.
- specs/047-test-timeout-expansion/deadline-inventory.json: classify new timing owners and intentional refusal clocks.
- scripts/release.py and receipt fixture tests: require nested terminal acceptance.
- tests/terminal_transport_test.py and cli_terminal_qualification_test.py: lifecycle, actual host terminal and mutation tests.
- docs/v1.0.1/development.md, CHANGELOG.md and this feature directory: truthful contracts, decisions and evidence.

## Design/Contracts

See [data-model.md](data-model.md) and [quickstart.md](quickstart.md). Internal evidence only; public CLI semantics remain unchanged unless a demonstrated defect requires correction.

## Delivery

Follow test-first transport/receipt implementation, actual host integration, full relevant checks and convergence. Commit/push/open an official focused PR; monitor final-head CI and review threads. Count the automatic initial Codex round and at most one explicit second round. Keep #147 open and stop before merging.
