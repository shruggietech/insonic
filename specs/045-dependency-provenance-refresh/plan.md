# Implementation Plan: Dependency refresh and provenance qualification

**Branch**: `codex/045-dependency-provenance-refresh` | **Date**: 2026-10-22 | **Spec**: [spec.md](spec.md)

**Tracking**: #168, milestone 29, Project 7; replacement for dependency PRs #157-#159.

## Summary

Adopt the three pending update sets on main 488f41f in one PR. Recompute the Go graph with pinned Go 1.27.1, retain pristine upstream sums required by the two maintained local desktop replacements, align capture tooling with the exact root Playwright pin, and use the existing full package/CLI campaign.

## Technical Context

**Language/Version**: Existing Node 22.12+, Python 3.12, Go 1.27.1; product remains 1.0.1.
**Primary Dependencies**: Playwright 1.64.1; go-winio 0.6.4, godbus 5.2.4, MinIO 7.4.1; Markdown-it 15.0.3, Next 16.4.2, site React/React DOM 19.3.1.
**Storage**: Existing manifests/lockfiles and compact canonical CI receipts.
**Testing**: Root checks/Node tests, Python unittest, site tests/build/export, affected native Go tests/vet, hosted full source/package checks.
**Target Platform**: Windows/Linux x86-64 CLI and desktop-with-CLI.
**Project Type**: Bounded dependency and maintainer-tool maintenance.
**Performance Goals**: Verified separate preparation; qualification 660s advisory target / 900s hard limit, 15s provisional reserve.
**Constraints**: No checksum bypass, native/brand or local patch upgrade, owner GUI, release, inference/model campaign or unrelated downgrade.
**Scale/Scope**: Eight direct package version changes across root, site and Go, plus required transitive resolution and two maintained upstream identities.

## Constitution Check

Before research: PASS. Maintenance preserves owner scope, CLI/runtime behavior, immutable inputs, supported adapters, hidden process guarantees and exact-source qualification.
After design: PASS. Local replacements and both provenance verifiers remain authoritative. No new public interface or constitutional exception is needed. Existing authorized publication supersedes the autopilot default pre-push pause; owner merge remains required.

## Project Structure

- Root and site package manifests/locks: requested JavaScript updates only. Desktop frontend React stays independently pinned.
- go.mod / go.sum: recomputed elected module graph; explicit pristine Wails/WebView upstream sums restored after tidy.
- scripts/capture-application-ui.mjs and tests/capture-application-ui.test.mjs: derive exact capture pin from root metadata and reject mismatched capture receipts.
- tests/go_module_notices_test.py and tests/webview-source.test.mjs: both maintained modules reject missing/mismatching or conflicting archive and module-file sums and accept exact restored identities.
- docs/v1.0.1/development.md: repeatable post-tidy upstream download/verification commands; CHANGELOG.md: concise maintenance note.
- This directory: spec, requirements/provenance checklists, plan, research, data model, quickstart and tasks. No new external interface requires a contracts directory.

## Decisions

1. Consolidate rather than rebase three bot branches: all updates share the existing exact-source package campaign; retain attribution and keep originals until qualification.
2. Select the exact versions requested, rather than latest-version discovery; resolve required transitives using the pinned package managers.
3. After tidy, use exact-version Go download JSON to independently verify the retained pristine upstream archive and module sums, then restore only those four exact entries. Explicit downloads did not persist sums for replaced modules in the verified Go 1.27.1 experiment. Keep existing verifiers and document this maintenance step instead of adding a parallel dependency framework.
4. Derive the capture Playwright requirement from the existing exact package pin, avoiding a second manually synchronized version constant. Current captures require that pin; frozen historical captures retain their original source records.
5. Leave six reviewer checklist prompts unchecked under the already authorized autopilot flow; author quality checklist passes 16/16. Unchecked reviewer prompts are not implementation completion claims.

## Verification and Handoff

Run focused red/green capture coherence and both-module checksum rejection regressions, then full root/site/Python and affected pinned Go checks through hidden noninteractive launchers. Full hosted CI must qualify both platforms, all Windows scopes, notices, maintained CLI ledger and source/run/attempt joins. Retain terminal phase timing and canonical artifact links. Resolve every received review and at most two Codex rounds; close superseded bot PRs only after replacement acceptance, then leave the final PR unmerged.
