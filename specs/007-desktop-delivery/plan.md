# Implementation Plan: S007 Desktop delivery

**Branch**: `codex/007-desktop-delivery` | **Date**: 2026-10-07 | **Spec**: [spec](spec.md)

## Summary

Deliver #10 through a static React/TypeScript Wails UI and shared runtime additions for settings, current cue projections and scoped playback. Build and qualify relocatable native GUI-with-CLI packages on the three already executed platform/architecture pairs. Existing date correction and work operations are reused.

## Technical Context

Go 1.27.1 and pinned Wails v2; React/TypeScript static bundle with governed React components, semantic CSS and exact radix-ui 1.6.7 where required. Existing SQLite/PostgreSQL and filesystem/S3 remain authoritative. Windows AMD64, Linux AMD64 and macOS ARM64 match pinned native artifacts and current CI. Node tests/typechecking, Go acceptance/races/vet, Python packaging tests, real media/Cueson/native webview and extracted-package qualification. No production Node server or model inference in CI. Native jobs remain bounded below ten minutes.

## Constitution Check

PASS before research and after design: CLI-first shared operations; no GUI-only core logic; originals/current embedded evidence preserved; alternative backends retain their shared contracts; no silent routing fallback or per-item approval; version remains 0.0.0; governed kit untouched; hidden Windows children; reviewed PR is merge boundary, no release. Public docs contain no numbered slice artifacts.

## Research and decisions

Research agents dispatched by speckit-plan: desktop_ui, desktop_runtime and desktop_packages. Decisions and alternatives are recorded in [research](research.md). No unresolved design unknown remains.

1. Existing date correction and real durable work operations are reused. Foundation timer jobs are not the production Jobs screen.
2. Promote tool configuration into runtime settings with per-section CAS and atomic publication; CLI uses the same handlers. Display selected storage/catalog/graph profiles and document their existing advanced configuration path. Changing an established workspace backend is migration work, not a settings-file overwrite; portable migration remains #14.
3. Project bounded semantic current cues with revision/digest fences. Preserve exact Cueson integer clocks as strings; expose validated seek seconds separately. Never parse the whole Cueson document into JavaScript numbers.
4. Runtime-issued playback handles materialize leased copied originals or identity-verified reference snapshots. Native bridge returns opaque ticket URLs, never paths. Each ranged request revalidates current media/recording identity. Bound handle count, inactivity and lifetime; keep leases alive while a playback handle is active and clean on expiration/close. Hosted native qualification demonstrated that the original same-origin Wails transport does not support required Linux/macOS media playback reliably. A process-lifetime HTTP listener binds only 127.0.0.1 on an ephemeral port, serves existing scoped tickets through the same handler, validates Host and browser Origin, exposes no CORS API and closes with the desktop. This replaces the original no-listener implementation decision without changing media limits or adding external routing.
5. Static React/TypeScript frontend uses AppFrame with Wails host profile, semantic fields/buttons/tables and current-authority refresh. Workspace/selection generations prevent stale replies overwriting current screens.
6. Packages include sibling CLI/GUI and pinned companions, complete support trees, exact inventory, hashes/notices/source provenance and matching help. Shared installed-companion defaults are used only when explicit workspace configuration is absent. No inference Python/weights are bundled or claimed.
7. Package construction and CI artifacts are authorized; official release/signing/notarization/promotion remain separate. Distribution must retain corresponding-source obligations for GPL companions rather than treating notices as sufficient.

## Project Structure

- `desktop/frontend/`: frontend dependencies, contracts/client, dedicated screens, governed styles, deterministic interaction tests and bundle build.
- `desktop/`: synchronized workspace bridge, native dialogs, opaque playback tickets/HTTP handler and integration tests.
- `cmd/insonic-desktop/main.go`: lifecycle/native picker/handler wiring and qualification.
- `internal/app/desktop_ops.go`, `internal/contracts/desktop.go`, runtime schemas and CLI: shared settings/playback/current-cue operations.
- `internal/packageenv/`: installed companion discovery and hash verification without persisted absolute installation paths.
- `scripts/package-desktop.py`, packaging tests and CI: platform layouts, inventory and extracted-package smoke.
- Public desktop/development/technology docs and capability matrix; root changelog.

## Execution and ownership

After tasks/analysis pass, existing research agents continue independent implementation with disjoint file ownership. Frontend agent owns desktop/frontend and index.html. Runtime agent owns shared app/contracts/CLI/schemas. Package agent owns scripts/package-desktop.py, tests, internal/packageenv and CI. Parent owns native bridge/main wiring, docs, integration and review/publication. Shared files and wire changes are coordinated before edits.

## Verification

Run npm check/test, frontend typecheck/interaction/build, Go acceptance/vet and affected races, packaging tests, public/offline exports, native client checks and extracted packages. Fixtures use committed audio/video with deterministic adapter callbacks. Qualification must verify visible screen workflows rather than merely bridge existence. Address every external review discussion, maximum two requested rounds. Record exact head checks in PR before owner handoff.

## Complexity Tracking

No constitution exception. No new database or durable assignment store. Playback registry is ephemeral leased access, not another media library or derived-run store.
