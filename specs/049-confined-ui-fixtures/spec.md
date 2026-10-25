# Feature Specification: Confined UI Fixtures

**Feature Branch**: `codex/049-confined-ui-fixtures`

**Created**: 2026-10-25

**Status**: Specified

**Input**: S049 for [#176](https://github.com/shruggietech/insonic/issues/176), maintainer UI fixtures must never write to the Windows drive root.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Confine temporary state (Priority: P1)

A maintainer runs application capture or documentation browser qualification and all disposable profiles, libraries and receipts remain under the designated temporary parent.

**Why this priority**: The existing Windows path selection writes to the drive root.

**Independent Test**: Exercise both tools with a nested temporary parent containing spaces and Unicode, then inspect the allocated location and its removal.

**Acceptance Scenarios**:

1. **Given** a valid nested temporary parent on Windows or Linux, **When** either tool allocates a fixture, **Then** the actual fixture is its strict descendant and never a volume root child chosen by discarding the temporary path.
2. **Given** an unusable or root-valued temporary setting, **When** allocation begins, **Then** the tool fails explicitly without creating a fixture at the drive root or silently selecting another location.

### User Story 2 - Release all owned state (Priority: P1)

A maintainer can rely on successful and handled failed runs to release their browser/helper processes and fixture state without deleting unrelated files.

**Why this priority**: Initialization failures currently escape cleanup responsibility.

**Independent Test**: Inject marker/profile initialization failures and cleanup refusal, then verify owned state removal, unrelated sentinel preservation and actionable residual diagnostics.

**Acceptance Scenarios**:

1. **Given** an allocated fixture, **When** marker creation or profile initialization fails, **Then** cleanup remains responsible for the newly allocated directory.
2. **Given** success, handled failure, or supported timeout/cancellation, **When** the run exits, **Then** owned children are stopped before fixture removal and successful cleanup is verified.
3. **Given** changed ownership or a locked residual, **When** cleanup cannot finish, **Then** the run fails with the owned residual location and reason; it never reports complete cleanup.

### Edge Cases

- Nested spaces and Unicode, Windows drive and UNC roots, relative/empty settings, nonexistent or unwritable parents, and symlink/junction aliases to roots.
- Marker write failure, altered/replaced fixture, partially initialized profile, repeated cleanup and failed child shutdown.
- Forced process termination cannot guarantee finalization; only supported handled cancellation and deadlines carry cleanup guarantees.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Both maintainer tools MUST use one consistent allocation policy under the actual designated temporary parent on Windows and Linux.
- **FR-002**: Allocation MUST reject root-valued, relative or unusable parents explicitly, without root fallback or arbitrary host cleanup.
- **FR-003**: Cleanup responsibility MUST start immediately after allocation, including failures before marker/profile initialization completes.
- **FR-004**: Cleanup MUST verify ownership and confinement before recursive deletion, preserve unrelated files and release owned child processes first.
- **FR-005**: Handled success, failure and supported deadlines/cancellation MUST remove owned state or report a failure with the residual location and reason. Successful receipts MUST require verified cleanup.
- **FR-006**: Regression checks MUST exercise actual filesystem allocation and cleanup failures, Windows path semantics, nested spaces/Unicode and unrelated sentinel preservation without creating drive-root fixtures.
- **FR-007**: Actual application and documentation browser qualification MUST run on Windows and Linux, preserve source/capture receipt authority and record allocated location and post-run absence. Any unexecuted platform MUST be disclosed.
- **FR-008**: Nearby allocation sites MUST be audited for the same root-selection defect; findings, proportional repairs and validation MUST be recorded in compact slice evidence and the changelog.
- **FR-009**: Existing hidden noninteractive process launches, browser isolation, functional checks and qualification timing contracts MUST remain effective.

### Key Entities

- **Temporary parent**: The designated absolute existing non-root directory under which disposable state may be allocated.
- **Owned fixture**: A unique newly allocated directory with retained identity and cleanup responsibility independent of successful marker creation.
- **Cleanup result**: Verified removal or a failure naming the owned residual and cause; it gates successful publication.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Both tools allocate and remove fixtures beneath the designated parent on Windows and Linux, with zero root-level allocations.
- **SC-002**: All injected initialization failures remove the allocated fixture; ownership changes preserve unrelated state; removal failures produce actionable residual diagnostics.
- **SC-003**: All required repository checks and applicable exact-head CI checks pass, and every review thread from at most two Codex review rounds is addressed before owner review.
- **SC-004**: Actual browser runs retain their source and capture receipts and prove post-run absence; qualification reports preparation, qualification and combined time under the existing acceptance contract.

## Assumptions

- This slice repairs disposable maintainer fixtures only. Broader #147 campaign work remains separate.
- Existing owner-deleted drive-root directories require no recovery or sweep. No arbitrary host directory deletion is authorized.
- Invalid temporary parents fail explicitly; there is no demonstrated need for a shortened Windows root path.
- User authorization covers pushing, publishing the PR, CI fixes and two review rounds. Merge remains owner-controlled.

## Clarifications

### Session 2026-10-25

- No critical ambiguity requires owner input. Functional scope, lifecycle, failures, dependencies, observability, constraints, terminology and completion signals are clear from #176 and the constitution; implementation choices proceed under autopilot.
