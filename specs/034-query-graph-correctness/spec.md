# Feature Specification: Query and graph correctness

**Feature Branch**: `codex/034-query-graph-correctness`
**Created**: 2026-10-17
**Status**: Ready for planning
**Input**: Run S034 under autopilot for #122, #123, #124, #125, #128 and #129; push, open a PR, resolve at most two review rounds, stop before merge.

## User Scenarios & Testing

### User Story 1 - Retrieve complete graph and search results (Priority: P1)

A user follows result cursors to retrieve every selected node and relationship, and continues a CLI search beyond its first page.

**Why this priority**: #122 silently loses cross-page relationships; #129 prevents normal continuation.
**Independent Test**: Compare the union of all pages with the selected graph and exhaust a search exceeding 500 matches.
**Acceptance Scenarios**:

1. Given a star graph whose center sorts after its leaves, when every page is consumed at limits 1, 2, 100 and 500, then every selected edge is returned without permanently omitted cross-page edges.
2. Given more than 500 search matches, when the user follows each returned cursor, then all matches are reachable through an executable CLI command, with continuation visible in human output.
3. Given a changed query or corrected corpus, when an old cursor is reused, then the request fails with a conflict rather than mixing result sets.

### User Story 2 - Execute advertised query contracts (Priority: P2)

A user applies advertised timezone forms and saves the query shown in contextual help.

**Why this priority**: #123 and #124 reject inputs advertised by contracts.
**Independent Test**: Exercise local, IANA and fixed-offset date filters and submit the saved-query example through schema validation and runtime dispatch.
**Acceptance Scenarios**:

1. Given local, America/New_York or +02:00, when query validation, execution and calendar filtering run, then all use the same interpretation as metadata dates, including DST boundaries.
2. Given query-save help, when fixture identities are supplied, then its example validates and saves; missing or blank titles fail the save-specific contract.

### User Story 3 - Run unambiguous native reads (Priority: P2)

A user submits exact typed parameters and receives actionable native grammar diagnostics.

**Why this priority**: #125 discards duplicate bindings and #128 obscures accepted grammar.
**Independent Test**: Reject duplicate names and nested members before execution, preserve large integer values, and exercise terminating semicolons, multiple statements and pure-function validation.
**Acceptance Scenarios**:

1. Given repeated parameter names or repeated members inside a parameter definition, when submitted, then the query is rejected before backend execution.
2. Given one trailing statement terminator, when the otherwise valid read runs, then it executes; multiple statements, mutation and effectful procedures remain rejected.
3. Given an unsupported function or grammar token, when rejected, then bounded diagnostics and contextual help name the supported restriction and correction without echoing query secrets.

### Edge Cases

Empty graph, self-loop, both edge directions, cross-page star, edge-heavy graph, byte limit, modified cursor, changed page size, stale corpus, unknown timezone, DST gap/fold, day precision, Unicode/blank title, duplicate nested JSON, exact signed integers, quoted/comment semicolons, multiple statements and unknown functions.

## Requirements

### Functional Requirements

- **FR-001**: All selected graph relationships MUST remain retrievable across bounded deterministic pages (#122).
- **FR-002**: CLI search MUST consume its continuation cursor and expose continuation before human output truncation (#129).
- **FR-003**: Query validation, execution and calendars MUST share metadata-date timezone resolution, including bundled IANA data on supported hosts (#123).
- **FR-004**: Query-save schema and help MUST require a nonblank title and provide a runtime-executable example (#124).
- **FR-005**: Native parameter files MUST reject duplicate members before decoding and preserve exact numeric values (#125).
- **FR-006**: Native reads MUST accept one trailing terminator, document the supported grammar/functions and report actionable bounded rejections without weakening read-only enforcement (#128).
- **FR-007**: Shared runtime behavior MUST remain aligned across CLI and desktop and both graph backends; existing authorization, result budgets and cursor invalidation MUST remain enforced.

### Key Entities

Query definition and typed parameters; deterministic continuation bound to query and corpus; hydrated result rows and referenced graph edges; selected calendar timezone; immutable saved-query version and title.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Complete page unions equal the selected graph at limits 1, 2, 100 and 500, and a search fixture greater than 500 rows can be exhausted.
- **SC-002**: Advertised timezone forms pass validation and produce expected query/calendar day membership, including DST and packaged Windows timezone availability.
- **SC-003**: Emitted query-save examples pass schema and live dispatch, while missing/blank titles and duplicate parameter JSON are rejected.
- **SC-004**: Native grammar regressions accept a single terminating semicolon while rejecting multiple statements and all existing prohibited effects.
- **SC-005**: Repository, affected product and site checks pass; exact-head CI and at most two review rounds are recorded before owner handoff.

## Assumptions

Scope is exactly the six authorized issues. #114 and #147 remain broader tracking work; provider/installer/desktop redesign defects are excluded. Product version remains 1.0.1 until a separate release. No migration or release is required. Push and PR publication are explicitly authorized; merge is reserved to the owner.

## Clarifications

### Session 2026-10-17

- Use opaque continuation through the existing result envelope; preserve existing cursor binding and bounded output rather than adding an unbounded graph response.
- Accept a single trailing terminator; preserve the existing read-only grammar and give actionable errors for unsupported constructs. No expansion to arbitrary functions is required.
- Save requires a nonblank title, matching the current runtime and documented saved-query identity.
