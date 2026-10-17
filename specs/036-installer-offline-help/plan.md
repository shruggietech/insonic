# Implementation Plan: S036 Installer progress and offline documentation

Branch: codex/036-installer-offline-help. Date: 2026-10-17. Specification: [spec.md](spec.md).

## Summary

Add native modeless engine progress and outcome-specific MSI pages; use a shared installed-help resolver and OS association opener for the installer and shared manifest validation for a controlled desktop help view. Preserve client launch/user boundaries and silent lifecycle checks. Qualify full transactions and actual native help activation on disposable hosted runners.

## Technical Context

Language: pinned Go, Python 3 and TypeScript/React. Dependencies: native MSI tables via ctypes, pinned Wails v2.16, existing Windows user-token launcher and package/browser qualification. No new runtime framework or storage migration.
Targets: Windows x86-64 MSI/CLI/desktop and Linux x86-64 desktop help. Existing SQLite/PostgreSQL, filesystem/S3 and LadybugDB/ArcadeDB contracts remain shared.
Constraints: 660-second CI budget, no visible local consoles, actual native evidence separated from static/dialog fixtures, installed-version offline help, non-elevated same-user launch, no inference or owner media.

## Constitution Check

Before and after design: PASS. Exact #109/#110/#111 scope; CLI/GUI share native help mechanisms; no media/data ownership change; no routing/policy expansion; native user boundaries preserved; public contracts separate from numbered planning; current version unchanged; no kit byte edits; explicit push/PR authorization honored and owner retains merge. No exceptions.

## Project Structure

- internal/launcher/: installed-help validation, Windows association/limited-token path and Linux xdg-open dispatch; focused tests.
- cmd/insonic-launcher/main.go: documentation mode and actionable bounded errors for elected interactive action.
- desktop/bridge.go and frontend/src/: native help action, one compact control, click/error/keyboard coverage.
- scripts/windows_msi.py: Progress dialog, subscriptions, operation/result pages, fresh-install independent documentation action.
- scripts/qualify-windows-msi.py and new UI helper: real full-UI transaction matrix and existing silent/ownership/rollback retention.
- scripts/package-desktop.py and native/frontend qualification: installed help activation and browser content/resource/navigation evidence.
- tests/: portable model, launcher and native harness regressions.
- docs/v1.0.1/desktop.md, windows-installation.md, screenshots and CHANGELOG.md: truthful final contracts/evidence.

## Delivery Strategy

Regressions precede changes. Shared help resolution first; independent MSI progress/result authoring next; wire desktop and completion elections, then real native qualification. Use existing clones, actor/context ownership and bounded process tracking. Full UI opens only on acknowledged disposable CI, never the owner's desktop. Run root checks/tests, Python suite, Go/native/vet, frontend and site checks, then exact-head hosted package/native/alternate-backend checks. Publish the three-issue PR, resolve at most two code/security rounds and audit all threads. Stop unmerged.

## Complexity Tracking

No constitution exceptions. Two read-only research agents inspect MSI and help boundaries under the planning command; primary agent owns implementation.

## Qualification corrections (2026-10-17)

A focused invocation trace replaces repeated inference from ignored MSI action status: actual arguments, startup desktop, UI-library initialization and delegate exit identify the boundary before changing dispatch. The disposable actor uses the CI Default desktop selected by MSI, with exact-account grants and complete restoration. The manual diagnostic cannot publish feature proof. Maintenance Repair uses standard omus without recaching its open package; native0 and exact restored bytes remain required. Deterministic writer-held regressions also correct single-table catalog lists and named model lookups to avoid unnecessary immediate writer reservations, retaining multi-table snapshots. These proportional corrections unblock the mandatory existing package journeys without reducing their assertions.
