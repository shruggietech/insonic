# Feature Specification: Managed worker readiness diagnostics

**Slice**: S048
**Feature Branch**: `codex/048-worker-readiness-diagnostics`
**Created**: 2026-10-25
**Status**: Implemented and qualified; unmerged for owner final review
**Tracking**: #174, bounded increment of #147; milestone 32. The broader audit stays open.

**Input**: Explain worker setup/readiness failures, measure cold and warm probe stages, and make nested verification respect an explicit enclosing budget. Deliver through a green reviewed PR; the owner performs the merge.

## User Scenarios & Testing

### User Story 1 - Understand failed worker inspection (Priority: P1)

A user preparing recognition or diarization can identify the failed prerequisite and corrective action instead of receiving only engine-unavailable.

**Independent Test**: Fail interpreter verification, launch, package discovery, import, inventory and output validation through shared application operations.

**Acceptance Scenarios**:
1. Given an import failure, inspection identifies the import stage and fixed reason without exposing exception text or protected inputs.
2. Given deadline expiry or caller cancellation, inspection distinguishes those outcomes from a missing or incompatible engine.
3. Given invalid or changed inventory, verification identifies inventory/integrity failure and refuses readiness or reuse.

### User Story 2 - Complete readiness within its real budget (Priority: P1)

A user checking both roles receives bounded, truthful readiness. Nested work cannot restart an expired budget or report unverified work as incompatible.

**Independent Test**: Exercise single-role setup and combined readiness with controlled slow prerequisites, earlier deadlines and cancellation.

**Acceptance Scenarios**:
1. Cold verification retains bounded observations distinguishing prerequisite verification, import and inventory cost.
2. An earlier caller deadline governs every nested operation and expiry is reported accurately.
3. Successful readiness/reuse of accepted environments still requires byte identity and substantive compatibility checks.

### User Story 3 - Assess an actual managed CPU environment (Priority: P2)

A maintainer compares cold/warm managed verification and a real CPU workflow using source-bound receipts.

**Independent Test**: Run failure fixtures and isolated managed CPU qualification without changing owner-installed software or libraries.

**Acceptance Scenarios**:
1. A controlled reproduction of ambiguous final-probe failure identifies its actual failed boundary after repair.
2. Cold/warm managed receipts bind source/environment identity, stage timings and actual outcome. Passing repetitions do not establish the historical S046 cause.

### Edge Cases

Expiry during interpreter hashing, before launch, import, inventory enumeration/hashing or receipt verification; unavailable executable, missing/incompatible package/device, corrupt/oversized/duplicate output, unreadable/changed dependencies and inventory limits; cancellation after exit, cleanup failure and sensitive child text; older receipts without new observations; one role failing while another succeeds.

## Requirements

### Functional Requirements

- **FR-001**: Failed inspection MUST expose a fixed reason, failed stage and correction for interpreter, launch, deadline, cancellation, package/version, import, inventory and output-validation failures.
- **FR-002**: Inspection MUST retain bounded monotonic observations distinguishing prerequisite, library and inventory cost for each role/mode.
- **FR-003**: Nested work MUST respect its original enclosing deadline and cancellation; expired work MUST NOT start a fresh budget.
- **FR-004**: Combined readiness MUST preserve truthful independent role outcomes and distinguish unverified/timed-out work from demonstrated incompatibility.
- **FR-005**: Compatibility, immutable environment identity, dependency hashing and user-selected routing remain authoritative. Diagnostics MUST NOT make failed verification eligible for readiness/reuse.
- **FR-006**: CLI and GUI MUST share diagnostics/readiness. Existing environment receipts without new optional fields remain usable when original integrity checks pass.
- **FR-007**: Diagnostic fields MUST use fixed labels and bounded numeric facts, excluding raw child output, exception text, secrets and arbitrary paths.
- **FR-008**: Controlled regressions MUST reproduce ambiguous boundaries and demonstrate classification/budget behavior, including negative integrity and cancellation cases.
- **FR-009**: Acceptance MUST include source-bound cold/warm managed CPU verification and actual CPU processing, with stage timings and unresolved historical cause explicit.
- **FR-010**: No owner profile/library mutation, installed-product replacement, new routing, release or merge is included. Other #147 dimensions remain tracked.

### Key Entities

- Inspection outcome: role/mode, verified compatibility, diagnostic reason/stage and correction.
- Stage observation: allowlisted stage/state, monotonic duration and bounded counts.
- Readiness budget: original expiry and remaining allowance for nested work.
- Environment receipt: accepted dependency identity and inspection; observations do not replace byte verification.

## Success Criteria

- **SC-001**: Every elected failure fixture yields its expected reason/stage; no sensitive marker escapes diagnostics.
- **SC-002**: Earlier-deadline/cancellation cases obey enclosing test budgets and reject readiness/reuse after verification failure.
- **SC-003**: Cold/warm managed runs retain source/environment identity and stage timings; actual CPU processing passes on the reviewed implementation.
- **SC-004**: Applicable checks pass and individual findings from at most two Codex rounds are resolved before handoff.

## Assumptions and Clarifications

The production probe/readiness allowance remains 60 seconds unless measurements demonstrate a concrete need for a documented adjustment. Test headroom is separate. Diagnostic fields may be additive; older receipt integrity/compatibility requirements remain. The historical S046 cause is unestablished; controlled reproduction proves a present mechanism rather than retrospective identity. Routine choices follow autopilot. Historical browser forensics and broader acoustic/GPU/external-service work are outside this increment.
