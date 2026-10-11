# Implementation Plan: Release workflow hardening and v1 launch preparation

**Branch**: `codex/016-release-workflow-hardening` | **Date**: 2026-10-10 | **Spec**: [spec.md](spec.md)

**Tracking**: [Issue #44](https://github.com/shruggietech/insonic/issues/44), v1 launch readiness milestone, insonic Project.

## Summary

Reuse native qualification for PR/release, isolate engine and desktop runtime work with their own immutable caches, build six package variants independently and retain exact-source collection. Add an explicit Pages transport with preflight and verified original published archive authority. Bundle the matching processing worker and elect extracted-package connected qualification. Preserve generic hosting, unsigned/ad-hoc signing disposition, full existing tests/timeouts and the owner merge boundary.

## Technical Context

**Language/Version**: Existing Python maintainer tools, Go 1.27.1 shared runtime, Node 22/Next.js docs and GitHub Actions.

**Primary Dependencies**: Pinned native libraries/media sources/Cueson, reusable GitHub Actions, existing release verifier and processing adapters.

**Storage**: Temporary owned extracts/workspaces; verified candidate archives; existing Pages gh-pages tree. Product storage backends unchanged.

**Testing**: Node workflow/release/site regressions, Python release/Pages/package/journey/native tests, affected product/site checks, actual six-variant CI and publication-disabled dispatch, controlled and elected Windows CPU packaged journeys.

**Target Platform**: Windows x86-64, Linux x86-64, macOS ARM64 packages; acoustic run Windows CPU.

**Project Type**: Shared CLI/desktop product with internal build/release tooling.

**Performance Goals**: Full warmed PR and release workflow turnaround under 600 seconds; two warm PR measurements and one complete release-path measurement.

**Constraints**: No product tag/release/deployment; no operator audio; verified source/attempt; hidden bounded children; unchanged acceptance/timeouts; no secret values in evidence; LF UTF-8 without BOM.

**Scale/Scope**: Three platforms, two package variants, two runtime lane families, two promotion transports and existing three-fixture connected journey.

## Constitution Check

Pre-research and post-design gates PASS:

| Principle | Design evidence |
| --- | --- |
| I owner scope | Prepare reviewed launch deliverables without new usage policies or actual publication. |
| II shared CLI | Extracted CLI and worker qualify the same shared operations; no GUI-only implementation. |
| III source identity | Full source/package/archive checks and exact current recovery comparisons; no operator library writes. |
| IV configured adapters | Pages/generic election; existing alternative storage/credential checks remain; no surprise route. |
| V truthful efficient development | Split independent jobs/caches, full warm measurements, honest mock/live and acoustic limits. |

## Project Structure

Internal artifacts are under `specs/016-release-workflow-hardening/` (spec, checklists, research, plan, data model, contracts, quickstart, tasks, analysis, convergence and evidence). Changes belong in `.github/workflows/{ci,native-qualification,release}.yml`, `scripts/{qualify,package-desktop,qualify-journey,release,pages}.py`, related `tests/`, `docs/v1.0.0/` authoritative release/development contracts and root `CHANGELOG.md`. No governed kit or v0.0.0 bytes change.

## Phase 0: Research

Three read-only research agents assessed CI/cache timing, existing Pages/signing and extracted-package journeys. [research.md](research.md) resolves all unknowns (D01-D10); no outstanding clarification exists.

## Phase 1: Design

1. Runtime becomes independent engine and desktop/credential lanes, aggregated under the current required native check. Each uses a distinct dependency/qualification-bound cache, separate from CLI/desktop package caches. Desktop builds its own native sibling CLI through a shared helper. Every previous scenario remains.
2. Reusable package lanes accept snapshot/release documentation target and explicit configured signing inputs. Release dispatch uses the same jobs and separate publication-target docs, awaits every required native/package job and keeps artifact matching by revision/attempt. Publication/deployment authority stays in separate trusted-main jobs.
3. Pages preflight validates existing repository, legacy source gh-pages:/, insonic.io, marker files and current environment branch policy. Verified publication promotion constructs a complete archive-derived tree, preserves CNAME/.nojekyll/history, creates a parented Git tree/commit and advances the ref without force. Explicit build request and bounded polling require exact resulting commit. No live deployment during this slice; isolated API tests and read-only preflight report their actual scope. Generic-host path stays separate.
4. The package includes fixed `companions/processing/processing-worker.py`, covered by existing inventory. Journey elects `--package`, receipt and exact revision, validates/extracts through existing package helpers, derives CLI/media/Cueson/worker only from that package and strips development loader overrides. Python environments and model snapshots remain explicit elected inputs. Controlled adapter remains visibly synthetic; real mode uses packaged worker.
5. Prepare concise highlights/launch instructions and observed signing status. After owner squash merge, rebuild from the new exact main SHA; review artifacts cannot become that candidate.

Contracts and validation entities are described in [data-model.md](data-model.md), [launch contract](contracts/launch.md) and [quickstart.md](quickstart.md).

## Complexity Tracking

No constitution exceptions. A built-in Pages transport avoids unnecessary credential provisioning; independent lane families remove measured serialization rather than altering product architecture.
