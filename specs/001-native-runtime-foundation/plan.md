# Implementation Plan: Native runtime foundation

**2026-10-07 dependency maintenance**: Maintained Cueson package/schema references now identify the required v1.2.0 contract. Historical qualification results describe their original session and do not establish later package execution; S005 records the exact current-version qualification.

**2026-10-07 dependency maintenance**: Maintained Cueson package/schema references now identify the required v1.2.0 contract. Historical qualification results describe their original session and do not establish later package execution; S005 records the exact current-version qualification.

**Branch**: `codex/s001-native-runtime-foundation` | **Date**: 2026-10-05 | **Spec**: [spec.md](spec.md)

## Summary

S001 delivers issues #1 and #2 together: a real workspace owner, shared CLI/Wails operations and exact native dependency qualification. Existing roadmap issue codes remain unchanged. Full durable catalog, artifact and credential implementations remain #3-#5.

## Technical Context

**Language/Version**: Go 1.27.1, Node 22+, Python 3.11+ Spec Kit.
**Primary Dependencies**: Wails v2.14.0, go-winio v0.6.2, flock v0.13.1, jsonschema v6.0.3, Cueson v1.2.0, Ladybug core v0.21.2 and immutable binding revision from research.
**Storage**: Atomic local configuration, OS advisory lock, session-only attempts.
**Testing**: Go security/concurrency/integration tests, three-OS native probes, existing Node tests and static export.
**Target Platform**: Windows, macOS and Linux; qualify runner architectures separately from artifact availability.
**Project Type**: CLI, on-demand runtime and Wails shell.
**Performance Goals**: One-MiB IPC, ten-second start timeout, finite child output, idle shutdown and CI jobs below ten minutes.
**Constraints**: Current-user IPC, no TCP listener, fixed redacted errors, hidden noninteractive children and immutable dependencies.
**Scale/Scope**: One writer per workspace, concurrent clients and bounded qualification fixtures.

## Constitution Check

Pre-research and post-design gates pass: approved scope; shared CLI/runtime/GUI behavior; all backend roles; secret isolation; actual native evidence; no usage restrictions/repeated approvals; internal planning separation; UTF-8/LF and unchanged brand bytes. No exception requested.

## Project Structure

```text
cmd/insonic/
cmd/insonic-desktop/
internal/app/
internal/workspace/
internal/runtime/
internal/process/
internal/contracts/
internal/qualification/
schemas/registry.go
desktop/
scripts/qualify.py
scripts/run-hidden.ps1
specs/001-native-runtime-foundation/
```

**Structure Decision**: Concrete foundation responsibilities; native graph/desktop checks use build tags; authoritative schemas embed offline.

## Decisions and implementation order

1. Canonical roots and OS advisory locks, never stale PID heuristics. Atomic no-replace initialization.
2. User-restricted Windows pipe DACLs and private Unix sockets. Exact protocol/workspace/request identity and bounded decoding.
3. Detached hidden runtime starts on demand; startup timeouts never steal ownership. Idle exit waits for requests and work.
4. Generation-fenced session attempts disclose restart loss; durable recovery is #3.
5. Fixed allowlisted diagnostics rather than arbitrary child text. Opaque secret references and no silent backend substitution.
6. CLI and Wails bridge share dispatch; local offline help; full desktop is #10.
7. Authorized push and official PR; all reviews resolved; at most a second Codex round; owner merge remains final.

## Verification gates

Discovery, alias/race/ownership/recovery, versions, request limits, local access, idle lifetime, disconnected work, stale attempts and redaction require meaningful tests. Native dependency/catalog/graph/GUI/help and alternative fixture checks run on the actual target OSes. Independent issue evidence prevents closing #1 while native execution is missing.
