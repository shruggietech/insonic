# Feature Specification: Desktop startup diagnostics and recovery

**Feature Branch**: `codex/042-startup-diagnostics`
**Created**: 2026-10-21
**Status**: Owner-authorized implementation in two independently mergeable increments
**Issues**: [#154](https://github.com/shruggietech/insonic/issues/154), with [#115](https://github.com/shruggietech/insonic/issues/115) limited to supporting private diagnostic-file behavior.

**Input**: When desktop startup fails, report an actionable cause, retain useful private diagnostics when possible, clean up owned resources, and permit a subsequent valid launch. Preserve existing runtime/profile policy and every native acceptance gate.

## Clarifications

### Session 2026-10-21

- Separate returned-error reporting and diagnostic-file creation from the subsequent concrete asynchronous startup/cleanup correction. Each increment may merge with its own passing required checks.
- An ordinary GUI launch needs visible failure feedback even without a console. Diagnostic-write and presentation failures must not replace the original startup cause.
- Subsequent valid launch means a new ordinary application process using the intended profile and normal directory permissions; this does not require repeated whole-application execution in one process.
- Private diagnostic records retain generated stage/category and numeric system error information, not arbitrary native/tool error strings, arguments, credentials, local paths or environment values.
- All historical startup signatures need not share one cause. Remaining historical criteria may keep #154 open after useful increments merge.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Understand a startup failure (Priority: P1)

A desktop user whose launch fails receives a visible cause and a useful corrective direction, with diagnostic-save status, even when no console is visible.

**Why this priority**: Discarded returned errors make a failed launch difficult to understand or report.

**Independent Test**: Inject a returned startup failure, capture user feedback and a private diagnostic record, then deny diagnostic creation and confirm the same original cause remains visible.

**Acceptance Scenarios**:

1. **Given** a returned startup error, **When** startup ends, **Then** the original cause and failed stage survive error wrapping and user presentation.
2. **Given** a consoleless ordinary launch, **When** startup fails, **Then** native failure feedback is attempted after owned application cleanup, independently of browser initialization.
3. **Given** unavailable diagnostic storage or presentation, **When** reporting runs, **Then** the primary failure remains the reported failure and the process exits unsuccessfully.
4. **Given** private diagnostics, **When** a record is retained, **Then** generated stage and numeric system-error metadata are useful while protected inputs and arbitrary error text remain absent.

### User Story 2 - Create private diagnostics in a writable directory (Priority: P1)

An ordinary Windows user can create a private diagnostic log and lock in a directory granting inherited Modify access, without granting unnecessary ownership-changing rights or elevating the application.

**Why this priority**: Failure to secure a newly created log can erase the only persistent explanation of a failed launch.

**Independent Test**: Exercise inherited Modify and explicit current-user access, inspect resulting owner/ACLs, and retain refusal of denied, foreign-owned, redirected or mismatching files and concurrent lock ownership.

**Acceptance Scenarios**:

1. **Given** a newly created file already owned by the current user, **When** its privacy is established, **Then** no ownership change or unnecessary ownership-changing access is requested.
2. **Given** an existing foreign-owned or redirected file, **When** diagnostic opening is attempted, **Then** it fails before command effects and does not take over or alter unrelated content.
3. **Given** existing logs, rotation and concurrent ownership, **When** logging runs, **Then** private access, identity and exclusive lock behavior remain enforced.
4. **Given** creation, privacy or lock contention failure, **When** the error is reported, **Then** corrective guidance distinguishes that boundary without exposing file content.

### User Story 3 - Finish failed startup and allow recovery (Priority: P1)

A startup failure completes as failure instead of terminating inside a callback or leaving the launch waiting. Owned partial initialization is cleaned up, and a later valid launch can use the intended profile.

**Why this priority**: Returned-error reporting alone cannot cover swallowed asynchronous errors or callback termination.

**Independent Test**: Inject synchronous and asynchronous environment/controller failures and partial initialization; assert one failure outcome, owner-thread cleanup, no success fall-through, and a subsequent valid binding/application launch.

**Acceptance Scenarios**:

1. **Given** a synchronous or asynchronous initialization failure, **When** it occurs, **Then** startup completes with its original stage/system cause and never enters the ordinary application loop as success.
2. **Given** a partially initialized browser, **When** startup fails, **Then** owned resources close and release once on their owning thread after callbacks unwind.
3. **Given** late completion after shutdown, **When** its callback arrives, **Then** it cannot re-open destroyed state.
4. **Given** a controlled failed launch, **When** a subsequent valid process uses the intended profile, **Then** normal startup and natural release succeed without retry loops, profile deletion or forced browser termination.

### Edge Cases

- Missing/denied diagnostic directories, concurrent diagnostic locks, unknown errors and unavailable GUI presentation.
- Failed or successful native status with a missing output, asynchronous errors, message-loop termination before readiness and failure after acquiring some interfaces.
- Existing private files, rotation, foreign ownership, path replacement, symlinks/reparse points and cleanup after failure.
- Ordinary launches retain their persistent profile; unattended qualification uses its existing explicitly owned profiles and hidden-console tooling.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Preserve returned startup errors and their wrapped underlying causes, distinguish failed stages, and exit unsuccessfully.
- **FR-002**: Ordinary GUI startup failure MUST attempt visible native error feedback independently of the browser; unattended qualification MUST retain noninteractive reporting.
- **FR-003**: Retain bounded private generated diagnostics when possible. Diagnostic-write, close or presentation failure MUST NOT replace the original startup error.
- **FR-004**: Diagnostic records MUST exclude arguments, inputs, credentials, endpoints, environment values, local path details and arbitrary tool/native error strings. Only allowlisted stages and numeric system-error metadata supplement existing generated fields.
- **FR-005**: Windows private-file creation MUST work with ordinary inherited Modify permissions without unnecessary ownership-changing rights, preserving required owner, restrictive ACL, file-identity and reparse checks.
- **FR-006**: Existing logs, rotation, denied access and concurrent locks MUST preserve privacy and fail-before-command-effects behavior, with actionable boundary-specific corrections.
- **FR-007**: Synchronous and asynchronous startup failures MUST complete once with an error; failed/missing outputs and interrupted initialization MUST NOT fall through to normal initialization or the main application loop.
- **FR-008**: Partial initialization MUST release owned browser/controller/environment/callback resources once on the owning thread after callback unwinding; late callbacks MUST NOT restore destroyed state.
- **FR-009**: A subsequent valid ordinary process MUST be able to launch using the intended profile and ordinary permissions. No automatic retries, silent profile deletion, forced browser termination, dependency upgrades or GUI-framework replacement are introduced.
- **FR-010**: Any necessary maintained dependency patch MUST retain its exact pinned upstream identity, explicit patch scope, checksums, source/notice qualification and rejection of arbitrary replacements.
- **FR-011**: Preserve every elected native/package/installer case, genuine input, natural release, exact-byte proof binding and native operation allowance. Qualification follows the 660-second advisory target, 900-second hard limit and existing phase accounting.
- **FR-012**: Deliver reporting/diagnostic-file behavior and concrete startup termination/recovery behavior as independently verified increments. Issue closure requires that issue's own criteria; no common historical cause is assumed.

### Key Entities

- **Startup failure**: Original error chain, generated failure stage and optional numeric system error.
- **User failure feedback**: Actionable cause, diagnostic-save status and visible native presentation when available.
- **Private diagnostic event**: Bounded generated fields, allowlisted stage and numeric system error; no arbitrary error text.
- **Startup outcome**: Pending, successful or failed; one terminal outcome controls whether normal initialization may continue.
- **Owned partial initialization**: Interfaces, callbacks and windows whose cleanup retains owning-thread and lifetime rules.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Returned-failure regressions preserve original error identity and cause in user feedback, including when diagnostic creation is denied or native presentation is unavailable.
- **SC-002**: An ordinary inherited-Modify directory supports private log/lock creation; explicit access, denied storage, existing/rotated logs, ownership/path protections and concurrent lock behavior pass focused Windows checks.
- **SC-003**: Privacy regressions show no protected input, local path or arbitrary native-error text in persisted startup events, while stage/system-code evidence remains.
- **SC-004**: Controlled synchronous/asynchronous and partial-initialization regressions produce one failed outcome, skip the ordinary loop, balance owner-thread cleanup and admit a later valid launch.
- **SC-005**: Each frozen increment passes required local/hosted checks and exact-main acceptance; real Windows package/native acceptance retains all existing cases and natural release.
- **SC-006**: Modified maintained dependency bytes and package notices pass exact-source authority checks against explicit upstream and patch manifests; arbitrary replacements remain rejected.

## Assumptions

- The existing private diagnostic logger and native platform presentation facilities are reused.
- The first increment improves returned-error paths; it does not claim callback paths are corrected until the second increment passes.
- A narrow patch to pinned framework startup propagation is permissible when existing hooks cannot return errors. It is not a replacement of the GUI framework.
- S041's final-source cold/warm campaign remains separately revision-bound and does not block these product merges.
- No release, unrelated dependency merge, owner-desktop exercise or discretionary hosted review campaign is authorized or required.
