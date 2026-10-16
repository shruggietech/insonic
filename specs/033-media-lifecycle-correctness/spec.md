# Feature Specification: Media admission, playback, and refresh correctness

**Feature Branch**: `codex/033-media-lifecycle-correctness`

**Created**: 2026-10-15

**Tracking**: [S033 milestone](https://github.com/shruggietech/insonic/milestone/17), [insonic Project](https://github.com/orgs/shruggietech/projects/7), issues #140, #141, #142, #143 and #145.

**Status**: Implemented; local verification passed; publication/review pending

**Input**: Run S033 under autopilot for #140, #141, #142, #143 and #145. Push and open the PR, resolve up to two review rounds, then stop for owner final review and merge.

## User Scenarios & Testing

### User Story 1 - Report accepted refreshes accurately (Priority: P1)

A user refreshes metadata while a reader holds an old report. The accepted update succeeds; obsolete bytes remain available until retirement can finish.

**Why this priority**: #143 reports failure after replacement metadata commits, misleading the operator about accepted state.

**Independent Test**: Hold a report lease across refresh, inspect revision/work, restart and retry, then release/expire the lease and prove eventual retirement without duplicate acceptance.

**Acceptance Scenarios**:

1. **Given** a live old-report lease, **When** refresh commits, **Then** work succeeds and obsolete report cleanup remains pending.
2. **Given** pending cleanup, **When** retention clears and reconciliation runs, **Then** only obsolete unreferenced bytes retire.
3. **Given** genuine storage failure or stale acceptance, **When** the operation runs, **Then** the actual failure stays distinguishable and current accepted authority is preserved.

### User Story 2 - Admit embedded subtitles and describe playable bytes (Priority: P1)

A user imports a supported embedded transcript and opens playback whose type matches the returned canonical audio or preview.

**Why this priority**: #142 blocks supported admission and #141 supplies an incorrect media type.

**Independent Test**: Import deterministic audio/video using exact packaged companions, accept supported embedded tracks, and independently inspect returned playback bytes.

**Acceptance Scenarios**:

1. **Given** embedded SubRip, **When** selected admission runs, **Then** current transcript text and timing are accepted.
2. **Given** other supported embedded formats, **When** selected, **Then** Unicode text and elected format remain valid.
3. **Given** canonical audio, preview or readable legacy media, **When** playback opens, **Then** its MIME, size and digest describe the supplied bytes and original observations remain preserved.

### User Story 3 - Import transcript batches and inspect acquisition failures (Priority: P2)

A user imports a transcript manifest with per-item targets and distinguishes remote deadlines, input limits and invalid content.

**Why this priority**: #140 blocks a documented batch workflow; #145 loses corrective failure causes.

**Independent Test**: Submit a two-record manifest without a global target, then exercise delayed/oversized loopback transcripts and inspect durable work after reload.

**Acceptance Scenarios**:

1. **Given** a manifest with per-item targets, **When** submitted without --record, **Then** valid items reach admission; positional single imports still require a target.
2. **Given** an expired transcript deadline or exceeded declared/streamed byte ceiling, **When** inspected through wait/show/results and reload, **Then** stable causes and corrections naming supported transcript flags remain visible in human/JSON output.
3. **Given** explicit cancellation, malformed content or failed replacement, **When** work terminates, **Then** causes stay distinct, temporary inputs disappear and prior accepted data remains intact.

### Edge Cases

- Live/released/expired leases, current references, restart after commit, stale writes and genuine storage failures.
- WAV-to-FLAC, mono canonical audio, video-to-canonical audio, previews and legacy media.
- SubRip, ASS/SSA, WebVTT and mov_text where supported by exact packaged tools.
- Missing per-item selectors, mixed positional/manifest input and incompatible global selectors.
- Declared/streamed oversize, exact-limit input, slow headers/body, truncation, invalid content and cancellation.
- Credentials and hostile response bodies cannot enter diagnostics.

## Requirements

### Functional Requirements

- **FR-001**: Accepted refresh MUST succeed when only leases or references defer retirement (#143).
- **FR-002**: Deferred cleanup MUST persist across release, expiry/restart and retry without duplicate accepted revisions (#143).
- **FR-003**: Genuine storage and stale-write failures MUST remain observable; unrelated conflicts cannot be blanket-suppressed (#143).
- **FR-004**: Supported embedded transcripts MUST extract using exact packaged companions and preserve selected text/timing (#142).
- **FR-005**: Playback MIME/size/digest MUST describe actual canonical/preview bytes, retaining original metadata and legacy readability (#141).
- **FR-006**: Manifest targets MUST come from each item; --record MUST be required only for single positional admission, with incompatible forms rejected (#140).
- **FR-007**: Remote transcript deadline/byte-limit failures MUST retain stable distinct causes, bounded explanations and corrections naming the elected transcript options (#145).
- **FR-008**: Causes MUST survive durable wait/show/results and reload, distinguish cancellation/invalid content and preserve failed-replacement authority (#145).
- **FR-009**: Diagnostics MUST exclude credentials, protected input, private content and arbitrary provider bodies.
- **FR-010**: Deterministic fixtures MUST extend existing installed-package qualification, use exact companions and owned cleanup, support spaces/Unicode paths and hide noninteractive Windows processes (partial #114).
- **FR-011**: Shared CLI/desktop behavior MUST preserve filesystem/S3 and SQLite/PostgreSQL contracts.
- **FR-012**: Required checks MUST preserve meaningful integrity coverage and the 660-second CI budget without inference/model downloads or attended owner gates.

### Key Entities

- **Accepted recording**: Current revision, canonical publication, source observations and optional current document.
- **Pending retirement**: Obsolete publication whose durable cleanup waits on leases/references.
- **Playback descriptor**: Returned-byte identity/type, separate from original-source provenance.
- **Acquisition failure**: Stable category/reason, elected bound and supported correction retained in durable work.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All five selected issue reproductions have passing focused and installed-package regressions.
- **SC-002**: Held leases cause zero false refresh failures or duplicate accepted updates; obsolete bytes retire when retention clears.
- **SC-003**: Every elected playback/subtitle fixture describes or preserves actual supplied content.
- **SC-004**: Manifest batches and remote failures preserve expected per-item outcomes, causes, exits and authority across reload.
- **SC-005**: Final PR checks pass; findings are resolved within two review rounds and work stops before merge.

## Assumptions and Scope

Existing canonical audio, durable work and cleanup contracts remain authoritative. No catalog migration or new cleanup subsystem is expected. Scope is #140, #141, #142, #143 and #145; relevant reusable fixtures contribute to #114 while #114, #130 and #147 remain open. Graph/evidence, general help/rendering and model/worker readiness stay separate. The current development version remains synchronized until a separate release. No merge, tag or release is authorized.

## Clarifications

### Session 2026-10-15

- Autopilot resolves routine choices from contracts: retention delays mean pending cleanup; genuine acceptance/storage failures stay distinguishable. No critical owner decision remains unresolved.
