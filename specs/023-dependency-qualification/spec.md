# Feature Specification: Consolidated dependency qualification

**Feature Branch**: `codex/023-dependency-qualification`

**Created**: 2026-10-13

**Status**: Implemented and qualified, awaiting owner merge

**Tracking**: [#84](https://github.com/shruggietech/insonic/issues/84)

**Input**: S023 consolidates Dependabot PRs [#77](https://github.com/shruggietech/insonic/pull/77), [#78](https://github.com/shruggietech/insonic/pull/78), [#79](https://github.com/shruggietech/insonic/pull/79), [#80](https://github.com/shruggietech/insonic/pull/80), [#81](https://github.com/shruggietech/insonic/pull/81), [#82](https://github.com/shruggietech/insonic/pull/82) and [#83](https://github.com/shruggietech/insonic/pull/83) under autopilot. Push and official PR are authorized. At most two review rounds; hand off after exact-head green CI. Merge remains with the owner.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Qualify updated delivery dependencies (Priority: P1)

A maintainer receives one reviewable update covering the pending delivery automation dependencies, with dependable cache reuse and complete source/package integrity evidence.

**Why this priority**: Every native package and release depends on these automation boundaries.

**Independent Test**: Qualify complete native package candidates and reject malformed, mismatched or incomplete retained artifacts.

**Acceptance Scenarios**:

1. **Given** the five automation update PRs, **When** consolidation is qualified, **Then** every requested target is included using immutable identities and all required delivery lanes remain effective.
2. **Given** verified cached sources or a cache miss, **When** qualification proceeds, **Then** reuse and reconstruction preserve source/compiler/recipe checks instead of assuming a cache hit establishes integrity.
3. **Given** mismatched digests, source revisions, attempts or missing package variants, **When** a candidate is inspected, **Then** it is rejected and no release/publication occurs.

### User Story 2 - Preserve runtime behavior with updated libraries (Priority: P1)

Users retain the same Windows/Linux CLI, desktop, local/connected backends and credential behavior after the requested runtime library update.

**Why this priority**: Updated libraries reach transport, credentials, terminal behavior and native execution.

**Independent Test**: Run runtime, credential, backend and native acceptance with the consolidated runtime dependencies.

**Acceptance Scenarios**:

1. **Given** the runtime-group PR, **When** libraries are updated, **Then** requested versions and resolved dependencies are synchronized without silently changing configured routes or supported platforms.
2. **Given** malformed input, denied credentials, cancellation or transport failures, **When** existing operations run, **Then** existing redaction, ownership and access-integrity requirements remain enforced.

### User Story 3 - Preserve portable documentation (Priority: P2)

Readers retain complete public and installed offline documentation after the requested renderer update, including historical links and matching schema resources.

**Why this priority**: Documentation is shipped inside every desktop package and checked during candidate assembly.

**Independent Test**: Build and inspect public/offline documentation and retain historical URLs, local assets and synchronized versions.

**Acceptance Scenarios**:

1. **Given** the documentation dependency PR, **When** a clean locked installation exports documentation, **Then** every expected page/resource/link remains available without a production server.
2. **Given** candidate versus release rendering or historical documentation, **When** exports are produced, **Then** status/download identities remain truthful and frozen published content is unchanged.

### Edge Cases

Bot rebases after consolidation begins; overlapping workflow edits; lockfile transitive changes; archive versus direct artifact upload; enforced download digest errors; cache misses/read-only cache access; stale source/attempt artifacts; missing package variants; changed renderer segment URLs; secret-bearing transport failures; native dependency availability; delayed external reviews.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Consolidation MUST cover all seven requested PRs: cache and cache/restore 6.1.0, upload-artifact 7.0.2, setup-node 7.1.0, download-artifact 8.0.2, Next.js 16.4.0, and runtime-group go-isatty 0.0.24, x/crypto 0.58.0 and x/net 0.60.0.
- **FR-002**: Automation dependencies MUST remain pinned to immutable commits with accurately matched version annotations; moving tags or bot rebases MUST NOT silently change the elected input.
- **FR-003**: Artifact transfer MUST retain explicit archive expectations, digest rejection, clean exact-source and same-attempt identity, complete four-package matrices, corresponding sources and signing disposition. No setting may downgrade a digest mismatch to a warning or bypass the project verifier.
- **FR-004**: Cache reuse MUST retain verified original-source and compiler/recipe/source identity boundaries. Missing or unavailable cache writes MUST NOT weaken acceptance; corrupt cached bytes MUST NOT become accepted input.
- **FR-005**: The runtime update MUST retain CLI/desktop behavior, supported Windows/Linux packages, all six backend choices, protected credentials, route elections, cancellation and ownership semantics. Existing security and backend tests MUST remain required.
- **FR-006**: Documentation MUST remain a complete static public/offline export with historical URL/resource retention, local assets, schema/changelog rendering and truthful candidate/release download status. Frozen 1.0.0 source documentation/schemas MUST remain unchanged.
- **FR-007**: Owned dependency declarations, lockfiles, qualification records, tests and authoritative dependency references MUST agree on the elected versions. Software/docs/master-schema release identity MUST stay 1.0.1 candidate; upstream dependency identity is independent.
- **FR-008**: The complete affected repository, runtime, frontend, site, live backend, native and package/candidate checks MUST pass. Ten-minute job limits and meaningful integrity checks MUST remain; ordinary checks MUST NOT invoke real-model inference.
- **FR-009**: The slice MUST use linked GitHub tracking and one official PR that maps every original update to its implemented outcome. Original bot PRs MUST remain open until the consolidated PR is merged; they can then be closed as superseded without independently merging their branches.
- **FR-010**: Every review finding MUST be answered and resolved, with no more than two review rounds. The final PR head MUST have green CI before owner handoff. Windows command/tool children MUST stay hidden and noninteractive.

### Key Entities

- **Dependency election**: Source PR, captured source commit, dependency name, previous/requested version and immutable resolved identity.
- **Qualification evidence**: Exact source revision/attempt, supported platform/variant, checks, retained artifact identities and observed limitations.
- **Consolidation record**: Tracking issue, slice code, seven source updates, official PR and supersession state.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Seven of seven requested updates have a traceable implemented outcome in one official PR.
- **SC-002**: Both supported platforms and all four package variants pass the complete qualification boundary without weakening any integrity or security check.
- **SC-003**: Every expected public/offline documentation page and local resource passes export/link validation; frozen published source content has zero changes.
- **SC-004**: All review threads are resolved after at most two rounds, and the exact handoff head has green checks and a clean checkout.

## Assumptions

- This is a dependency qualification slice, not a product feature, release, deployment, toolchain replacement or installation-policy change. Managed workers, first-run setup and MSI remain their existing separate issues.
- All seven requested versions are available as recorded by the original PRs; immutable inputs and compatibility are verified during research.
- Source changes needed for these dependency interfaces are in scope, but unrelated maintenance is not.
- Bot supersession occurs after the owner squash merge. No independent bot merge, release, tag or deployment is authorized.

## Clarifications

### Session 2026-10-13

- Q: Which dependency versions are elected? A: The exact seven requested PR targets, not later/latest versions. Capture original commits before implementation and resolve immutable upstream identities.
- Q: When are original bot PRs retired? A: After owner merge of the consolidated PR; keep their tracking intact during qualification.
- Q: Does this publish a product release? A: No. Software/docs/schema remain 1.0.1 candidate, and the run stops for owner final review and merge.

Coverage scan: functional scope, artifact identities/state, failure/recovery, compatibility/security, performance, external dependencies, terminology and completion are Clear. New end-user interaction/localization is not applicable to this dependency-only scope. No critical ambiguity requires owner input.
