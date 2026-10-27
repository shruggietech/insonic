# Feature Specification: Bounded documentation publication

**Feature Branch**: `codex/054-bounded-pages-publication`

**Created**: 2026-10-27

**Status**: Specified

**Input**: S054 for GitHub #189. Automatically deliver a reviewed PR; the owner retains merging.

## User Scenarios & Testing

### User Story 1 - Publish complete growing documentation (Priority: P1)

A maintainer publishes the original accepted release export, including historical routes, without manually repairing a request that exceeds the hosting service's limits.

**Why this priority**: The latest publication required a manual transport workaround.

**Independent Test**: Publish a representative export exceeding the failed single-request size through an isolated hosting fixture and compare every final path and byte identity.

**Acceptance Scenarios**:

1. **Given** an accepted export larger than 46.7 MB, **when** it is staged, **then** bounded requests produce exactly its complete file inventory before publication.
2. **Given** unchanged content, a large individual text file, Unicode and binary content, **when** staged together, **then** each original byte identity and historical route is preserved.

### User Story 2 - Refuse incomplete or competing publication (Priority: P1)

A maintainer can trust that failed staging or a competing update cannot expose a partial export or overwrite another publication.

**Why this priority**: Existing release and destination authority must survive a transport repair.

**Independent Test**: Inject staging failures, incorrect content identities, deadline exhaustion and competing branch updates; verify publication is refused at the appropriate boundary.

**Acceptance Scenarios**:

1. **Given** failed or incomplete staging, **when** publishing, **then** the public branch and build remain unchanged.
2. **Given** a competing branch update, **when** publishing, **then** no forced overwrite or build request for the wrong commit occurs.
3. **Given** many staging operations, **when** the original time allowance is exhausted, **then** later requests stop without renewing the enclosing allowance.

### User Story 3 - Diagnose hosting failures (Priority: P2)

A maintainer receives bounded useful failure context without exposing credentials, signed URLs or submitted document contents.

**Why this priority**: The previous generic failure concealed the oversized request cause.

**Independent Test**: Inject structured, malformed, oversized and credential-bearing error responses and inspect the emitted diagnostics.

**Acceptance Scenarios**:

1. **Given** an unsuccessful hosting operation, **when** its error is reported, **then** method, query-free endpoint, status, request size and bounded redacted server explanation identify the operation.

### Edge Cases

- Empty files, repeated content, obsolete nonhistorical files, deep paths and text whose encoded request size greatly exceeds its raw size.
- Wrong blob identity, omitted/extra/duplicate tree entries, wrong mode or truncated inventory.
- Lost update response, failed or mismatched build, and a stale previous completed build.
- Missing publication authority, unsafe archive members, domain changes and missing historical counterparts remain refused.

## Requirements

### Functional Requirements

- **FR-001**: Publication MUST stage the complete existing bounded export without a single combined content request proportional to its total size.
- **FR-002**: Every request MUST have an explicit size bound; large individual files and binary files MUST preserve their exact bytes through an appropriate bounded transfer.
- **FR-003**: The final staged inventory MUST match every expected path, regular-file mode and full content identity before creating or advancing the publication commit; extra, missing, duplicated or truncated content MUST fail.
- **FR-004**: Existing trusted-main, release, original-asset, domain, environment, historical-route, non-forced branch and exact-build/live verification requirements MUST remain effective.
- **FR-005**: Intermediate staging failure MUST leave the branch unchanged; a concurrent update MUST not be overwritten. A lost update response MAY reconcile only against the exact elected commit.
- **FR-006**: Staging, branch update and build waiting MUST share one original enclosing deadline, with request waits bounded by remaining time.
- **FR-007**: Hosting failure diagnostics MUST include method, query-free endpoint, status, request size and a bounded redacted server explanation. They MUST exclude credentials, authorization headers, signed query values and submitted file bodies.
- **FR-008**: The repair MUST preserve existing release interfaces and immutable released content; verification MUST use isolated destinations and MUST NOT republish the live site or release.

### Key Entities

- **Accepted export**: Immutable bounded mapping of confined paths to original bytes.
- **Staged inventory**: Unpublished complete regular-file paths, modes and content identities.
- **Publication destination**: Existing elected repository, branch, domain, environment and observed head.
- **Operation diagnostic**: Bounded credential-free context associated with one failed hosting operation.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A representative export with at least 1,110 files and 54 MB completes isolated staging with 100% inventory and byte-identity agreement.
- **SC-002**: Every injected staging/inventory failure leaves the public branch unchanged; competing updates are never forced away.
- **SC-003**: A large individual text file, Unicode text and binary content retain exact original identities, while no operation renews the enclosing deadline.
- **SC-004**: Every unsuccessful hosting-operation test identifies its method, endpoint, status and request size while containing none of the injected credentials or query secrets.

## Assumptions

- Existing archive bounds and supported hosting destination remain unchanged. No product version bump, release rebuild, live deployment or native installer run is needed.
- Transport selection and request sizing are routine engineering choices under autopilot. The issue's existing operational workaround establishes publication, not a permanent source repair.
- No critical ambiguity requires owner input. All scope, data, flow, reliability, confidentiality, dependency, failure and completion categories are clear; transport implementation details belong in the plan.
