# Implementation Plan: Package dependency integrity

**Branch**: `codex/024-package-dependency-integrity` | **Date**: 2026-10-13 | **Spec**: [spec.md](spec.md)

**Input**: S024 objectives #86, captured Dependabot #87, Mermaid/KaTeX audit remediation; tracking [#88](https://github.com/shruggietech/insonic/issues/88).

## Summary

Qualify notices against the exact dependency versions linked into packaged executables, reject missing or inconsistent upstream evidence, and exclude unused tooling modules. Consolidate x/net 0.61.0 and elect Mermaid 12.2.0 with KaTeX 0.18.2 while retaining offline documentation and diagram behavior. The design uses a focused scripts/go_module_notices.py helper for assembly and archive membership checks, plus a hidden headless export renderer qualifier. See [research.md](research.md), [data-model.md](data-model.md) and [quickstart.md](quickstart.md).

## Technical Context

**Language/Version**: Go 1.27.1, Python 3, Node 22+, TypeScript/React, current unreleased version 1.0.1.

**Primary Dependencies**: x/net 0.61.0 elected; Mermaid 12.2.0 and KaTeX 0.18.2 elected; Wails and LadybugDB pins unchanged.

**Storage**: Existing inventory JSON and copied upstream notices in four Windows/Linux CLI/desktop archives. No operational database changes.

**Testing**: Python unittest, Node repository checks, Go test/vet, frontend tests, site/offline builds, hosted native adapter/package qualification.

**Target Platform**: Windows/Linux amd64, CLI and GUI-with-CLI; static portable docs.

**Project Type**: Internal package integrity and dependency maintenance in a CLI-first desktop application.

**Performance Goals**: Existing CI below ten minutes, bounded subprocess deadlines and reused caches; collect only shipped module evidence.

**Constraints**: Hidden noninteractive Windows child processes; exact clean source revision and workflow attempt; no release/deploy/merge; no new public inventory shape or frozen 1.0.0 edits.

**Scale/Scope**: #86 + #87 + one KaTeX advisory in the docs renderer. Research resolves executable metadata, selective source lookup, effective replacements, KaTeX options regression and hidden browser qualification.

## Constitution Check

Pre-research gate: PASS. Owner-approved scope is preserved, core behavior and six backends stay aligned, notice evidence is checked rather than inferred, documentation/version/publication boundaries remain intact, and all Windows tooling uses the verified hidden launcher. Internal changes require no public API expansion or constitution exception. Post-design gate: PASS. No public interface, operational state, version, frozen artifact, native pin or product policy change. No contracts directory is needed for this internal build/qualification slice.

## Project Structure

`specs/024-package-dependency-integrity/` contains specification, checklists, research, plan, data model, quickstart, analysis, tasks and final evidence. Source targets are `scripts/package-desktop.py`, `scripts/release.py` if affected inventory checks require it, focused `tests/`, `go.mod`, `go.sum`, `site/package.json`, `site/package-lock.json`, `site/scripts/`, current `docs/v1.0.1/` development/release contracts and `CHANGELOG.md`.

## Complexity Tracking

No constitution violations or new services, databases or public interfaces are proposed. All design unknowns are resolved; all requirements have planned meaningful checks.
