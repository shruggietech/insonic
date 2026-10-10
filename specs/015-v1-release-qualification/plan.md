# Implementation Plan: v1 release candidate and final qualification

**Branch**: `codex/015-v1-release-qualification` | **Date**: 2026-10-10 | **Spec**: [spec.md](spec.md)

**Input**: S015, [issue #42](https://github.com/shruggietech/insonic/issues/42), milestone v1.0.0 release candidate and [dependency PR #41](https://github.com/shruggietech/insonic/pull/41).

## Summary

Correct candidate/version preparation, preserve baseline identities, qualify connected fresh-workspace behavior and elected real engines, prepare 1.0.0 and collect six real native variants with matching documentation in unprivileged CI. Finish at owner PR merge handoff without a tag, public product release or deployment.

## Technical Context

**Language/Version**: Existing Go 1.27.1, Python maintainer/worker tools, Node 22, TypeScript/React and static Next.js.

**Primary Dependencies**: Current Cueson/media/Wails/graph pins; go-winio 0.6.3 and x/sys 0.49.0; installed pinned faster-whisper/Community-1 for elected local CPU execution.

**Storage**: Full existing filesystem/S3, SQLite/PostgreSQL and LadybugDB/ArcadeDB support.

**Testing**: Node/schema/text, Python release/source/harness guards, full Go/vet/affected races, frontend/site, connected product journey and separate elected acoustic measurements.

**Target Platform**: Windows x86-64, Linux x86-64 and macOS ARM64, both CLI-only and desktop.

**Project Type**: CLI-first shared runtime/desktop with versioned documentation and maintainer release tooling.

**Performance Goals**: Full cached CI including collection below ten minutes with all existing checks. Elected engines use explicit duration/thread/output/wall limits and measured RSS.

**Constraints**: No acoustic inference/download in required CI; hidden noninteractive children; immutable old schemas/docs; exact source and attempt identities; candidate collection has no secrets/write permission; no implicit provider/device fallback.

**Scale/Scope**: One connected fresh library/recovery journey, current Windows CPU acoustic measurements, six native package variants and a 1.0.0 candidate. Actual pyannote-profile enrollment is not base-weight retraining; generic training contracts remain deterministic.

## Constitution Check

Pre-research and post-design PASS: preserve owner scope, configured routing, CLI/shared GUI contracts, source identity/timing, full backend alternatives and truthful qualification/publication states. Correct poor preparation logic proportionally. No blanket cold-bootstrap exception is adopted.

## Project Structure

```text
specs/015-v1-release-qualification/   # intent, design, tasks and evidence
scripts/release.py                   # preparation and candidate collection
scripts/qualify-processing.py        # elected engine measurements
scripts/qualify-journey.py            # connected product qualification
tests/release_test.py                # history/version/source failures
tests/qualify_journey_test.py         # election/isolation/receipt failures
.github/workflows/ci.yml             # same-run docs and candidate artifacts
.github/workflows/native-qualification.yml # complete independent package/runtime lanes
internal/app/ internal/voicemodels/  # proportional corrections exposed by qualification
docs/v1.0.0/ schemas/v1.0.0/          # new current authorities, preserved old versions
site/ go.mod go.sum                 # candidate status and exact dependency updates
```

**Structure Decision**: Reuse existing product operations and verifiers. The maintainer harness coordinates actual CLI calls, not a new scheduler or owner gate. Candidate source closure compares with the selected checkout rather than accepting self-consistent stale receipts.

## Delivery Phases

1. Specify/clarify, quality review, read-only plan research agents, design/tasks and blocking analyze.
2. Regression-first corrections for candidate status/history/current source closure; integrate PR #41.
3. Connected controlled qualification and separate actual elected local CPU engines using current code and existing model bytes.
4. Prepare 1.0.0; audit all documentation technical vocabulary/glossary, current status and frozen baseline authorities.
5. Retain successful same-run real package/docs archives and collect a read-only candidate. Independent package calls overlap collection with full runtime/credential/bridge checks; every original check remains.
6. Local parity, converge, push/official PR, at most two review rounds, final exact-head green checks and owner handoff.

## Decisions and Measurement

[research.md](research.md) records D01-D14. CI artifacts identify the actual clean PR merge SHA, with branch head separately reported. Squash merge requires a new exact-main release candidate. Publisher identity/deployment settings are currently absent; report that disposition without provisioning them. Actual local paths stay in ignored harness configuration.

## Complexity Tracking

No constitutional deviation accepted. Measure full cached timing including aggregation; the prior slice's specific source-bootstrap exception does not waive future regressions.
