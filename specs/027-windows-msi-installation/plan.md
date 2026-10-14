# Implementation Plan: Windows MSI and installation guide

**Branch**: `codex/027-windows-msi-installation` | **Date**: 2026-10-13 | **Spec**: [spec.md](spec.md)

## Summary

Complete #65/#64 through one selectable-client Windows MSI, transactional maintenance, separately typed release identity, accurate installation guidance and autonomous real UI captures. Keep the four archive variants unchanged. Per-user installation owns program files/integrations, while user libraries/settings remain outside installer ownership.

## Technical Context

Existing Go 1.27.1 runtime/CLI, Python packaging and React/TypeScript desktop. Native Windows Installer database APIs plus Windows cabinet tooling author MSI; no additional installer tool EULA is accepted. Headless Chromium renders unchanged built frontend with a capture-only bounded bridge proxy to an isolated current CLI/runtime. Native authored installer captures and installation qualification run only in disposable CI runners. No native installer UI runs on the owner host.

Storage: MSI tables and embedded cabinet, payload inventory/receipt, existing profile/library state, versioned public PNGs and capture receipt. Tool dependencies are explicitly pinned or their actual platform/browser identity recorded. Capture binds exact source/version and fails on renderer/landmark errors. Tests cover MSI table relationships/ownership, actual installation lifecycle in CI, launcher quoting, format/source identity, capture isolation/provenance and online/offline asset parity.

Performance: reuse Windows desktop payload and native source cache; parallel fixture/capture and lifecycle checks; candidate completeness retains full gates within ten minutes. No model inference or user interaction is required.

## Constitution Check

Pre-design and post-design: PASS. CLI/shared runtime remains authority; GUI-only client selection retains shared bytes. No credentials/user data in installer ownership. No new routing/review gate. Brand assets remain governed. Current software/docs/schema candidate stays 1.0.1. Release/merge remain owner actions, including explicit final launcher wording signoff. No security/tenancy check is weakened.

## Project Structure

- scripts/windows_msi.py, package-msi.py: portable deterministic MSI model/validation, native database/cabinet generation and trusted receipt verification.
- scripts/qualify-windows-msi.py, capture-installer-ui.py: disposable-runner lifecycle and owned-process native UI capture.
- cmd/insonic-launcher/, internal/launcher/: GUI subsystem launcher, useful selected-client launch and noninteractive describe/preflight contract.
- scripts/package-desktop.py: include launcher/session assets in existing qualified payload.
- scripts/capture-application-ui.*: headless compiled application, bounded current-runtime bridge and source/image receipts.
- scripts/release.py, schemas/v1.0.1/release-candidate.schema.json: separate MSI identity plus unchanged archive matrix.
- site/lib/release-status.mjs, site/scripts/ and docs/v1.0.1/windows-installation.md: download and screenshot parity.
- .github/workflows/: exact-source MSI/capture qualification, candidate completeness and retained receipts.

## Decisions and sequence

2026-10-13: Per-user LocalAppData Programs installation is the bounded default. All-users is not added; direct MSI has no network acquisition, no global PATH integration and no prerequisite bootstrapper. Missing WebView2/VC++ uses official remediation. Native launch integrations are optional; desktop dependency semantics are explicit.

2026-10-13: Maintained WiX releases/tooling terms were researched. Choose native Windows Installer table authoring and cabinet tooling to avoid introducing commercial tool acceptance. Stable deterministic component/upgrade identities, standard transactional actions and tests remain required.

2026-10-13: One MSI contains selectable desktop/CLI integrations plus mandatory shared core. A Windows GUI-subsystem launcher opens the desktop or a persistent standard Windows command session with help/setup guidance. Interactive CLI launch is the user's elected product action; unattended qualification uses describe/preflight and never opens a console.

2026-10-13: Preserve four strict archive receipts and add one typed windows-installer-receipt plus installers list. MSI verification links to exact desktop payload inventory, corresponding source closure and help/license identity; install success alone never proves those bytes.

2026-10-13: A native MSI preview probe incorrectly reached the default desktop, and an indiscriminate window-close watchdog affected unrelated user windows. That launcher is permanently retired. No local native UI preview, desktop enumeration or window-close cleanup is authorized. Local UI captures use verified headless Chromium. Native MSI UI and lifecycle run only on disposable CI runners; cleanup targets exact owned process trees and never unrelated windows. This is a technical correction to execution isolation, not a reduction of screenshot scope.

2026-10-13: Application captures serve unchanged compiled UI with an isolated fixture and real bounded CLI/shared contracts. Renderer receipts distinguish headless browser from native webview. Policy-dependent Windows/UAC/reputation prompts that do not occur are documented from current primary guidance rather than fabricated screenshots.

2026-10-13: Proposed labels: Start folder insonic, GUI shortcut insonic, CLI shortcut insonic CLI, desktop shortcut insonic; final checkbox Launch insonic when desktop is selected and Open insonic CLI when CLI-only. These remain proposals pending explicit owner final-review signoff before shipping; concrete authored controls and images accompany review.

## Implementation phases

1. Test deterministic MSI/payload/ownership and safe launcher/capture contracts.
2. Build selectable-client installer, useful launcher and lifecycle handling.
3. Add exact format/source release integration and isolated capture tooling.
4. Publish version-matched text and real screenshots, with conditional OS guidance.
5. Run configured parity and native CI, converge, publish official PR, address at most two rounds and hand off after exact-head green checks.

## Complexity Tracking

No constitution violation or new product service. Capture-only bridge is necessary to render the actual shared application without an owner-visible native window. Native CI qualification is required because actual MSI UI cannot be safely probed on the owner's desktop.
