# Feature Specification: Release workflow hardening and v1 launch preparation

**Feature Branch**: `codex/016-release-workflow-hardening`

**Created**: 2026-10-10

**Status**: Specified

**Input**: Owner-authorized S016 hardens release/CI, exercises publication-disabled exact-source candidate assembly, prepares existing insonic.io Pages promotion, records signing disposition and verifies extracted packaged binaries. Push and official PR publication are authorized; at most two review rounds; stop at owner final review and squash merge.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Receive timely complete release evidence (Priority: P1)

A maintainer obtains six exact-source native package variants and matching publication-target documentation through the real publication-disabled release workflow.

**Why this priority**: Release jobs retain slower paths than PR qualification; a launch requires actual release-path evidence.

**Independent Test**: Assemble six packages, verify source/attempt/version/inventories and measure full warm PR/release runs.

**Acceptance Scenarios**:

1. **AC-001**: Given a clean reviewed commit, when publication-disabled release qualification completes, then six matching source-complete packages and public/offline release-target documentation form a validated candidate with no tag, public release or deployment.
2. **AC-002**: Given warmed verified caches, when full PR and release qualification execute, then each finishes below ten minutes without removed checks or enlarged timeout bounds.
3. **AC-003**: Given failed, stale, missing or mismatched platform inputs, when collection runs, then it refuses mixed-source/attempt or incomplete assets while independent successful lanes retain their receipts.
4. **AC-004**: Given native CLI qualification, when acceptance runs, then it requires real native graph behavior and rejects fallback execution.

### User Story 2 - Prepare the existing documentation destination (Priority: P1)

A maintainer can elect verified publication promotion to the existing insonic.io GitHub Pages site, preserving historical routes/domain and explicit signing/publication state.

**Why this priority**: Existing hosting and the generic promotion contract are not connected; a new host/credential is unnecessary.

**Independent Test**: Validate promotion against an isolated destination and mocked remote authority, rejecting invalid inputs before destination mutation.

**Acceptance Scenarios**:

1. **AC-005**: Given a verified published release and existing destination, when Pages promotion runs, then it installs original verified public documentation, preserves domain/historical routes and reports success only for the exact staged build commit.
2. **AC-006**: Given unpublished/corrupt assets, untrusted source or unexpected destination/domain, when preflight/promotion runs, then it fails before destination mutation.
3. **AC-007**: Given elected Pages or generic hosting, when promotion runs, then only that route executes and deployment credentials are absent from candidate builders.
4. **AC-008**: Given absent publisher identities, when readiness is reported, then unsigned/ad-hoc/unnotarized state remains explicit; configured signing failures abort before final hashes.

### User Story 3 - Verify the delivered binaries (Priority: P1)

A maintainer exercises extracted installed CLI/tools/worker through import, processing, speaker enrollment/matching, graph queries and disconnected-source restore in disposable owned workspaces.

**Why this priority**: Source-built journeys do not prove extracted package completeness.

**Independent Test**: Controlled and elected Windows CPU packaged journeys retain exact package/runtime/model identities and accepted authority comparisons.

**Acceptance Scenarios**:

1. **AC-009**: Given an extracted verified package, when a journey executes, then application commands use its installed CLI/native companions and real processing uses its matching worker, without source rebuild or ambient fallback.
2. **AC-010**: Given pinned local engines and licensed fixtures, when elected actual Windows CPU processing runs, then receipts identify package/model/tool/device and exact import/query/restore comparisons with honest observed outcomes.
3. **AC-011**: Given corrupt inventory, stale version, missing worker/tool or failed operation, when qualification runs, then it rejects boundedly without operator-media changes or substituted bytes.
4. **AC-012**: Given completed qualification/review, when handoff is assembled, then it includes concise highlights and exact clean-source commands, separating tested candidate behavior from signing/deployment configuration and authorized publication.

### Edge Cases

- Cold source builds/queues are reported separately; they do not excuse repeatable warm bottlenecks.
- A squash merge requires a new exact-main candidate; synthetic PR merge assets cannot become its release.
- Missing platform inputs cannot be replaced with earlier attempts.
- Release-target rendering does not publish downloads.
- Promotion preserves CNAME/.nojekyll and historical routes; concurrent destination updates must not be overwritten.
- Unconfigured signing differs from a failed configured attempt; neither becomes a false publisher claim.
- Unknown matching and editorial character counts do not establish independent acoustic accuracy.
- Qualification confines bounded hidden children, temporary homes/workspaces and model outputs to owned paths.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Retain every native, credential, bridge, backend, source/inventory and documentation check while reusing verified caches and independent runtime/package lanes.
- **FR-002**: Validate both package variants on Windows x86-64, Linux x86-64 and macOS ARM64 with one clean source/attempt/version and matching sources/publication-target documentation.
- **FR-003**: Full warm PR and publication-disabled release qualification MUST finish below 600 seconds without weakened checks or larger timeouts.
- **FR-004**: Native CLI acceptance MUST require actual native graph availability; generic lightweight qualification remains usable without native assets.
- **FR-005**: Pages promotion MUST target the existing repository/domain, install original verified published documentation, preserve existing domain and historical archive routes, and verify the exact deployed build commit.
- **FR-006**: Publication/source/inventory/archive/destination checks MUST precede mutation; candidate builders receive no deployment credentials or privileged publication authority.
- **FR-007**: Explicit generic-host selection remains supported without silent alternative routing.
- **FR-008**: Record observed publisher signing/notarization disposition; configured signing precedes hashes and failures abort.
- **FR-009**: Packaged journey qualification uses extracted selected CLI/native tools/worker, rejects incompatible/missing bytes and preserves exact current authority through disconnected-source restore.
- **FR-010**: Actual acoustic processing remains elected maintainer work outside deterministic CI; retain honest unknown/accuracy/device/platform limitations and bounded diagnoses.
- **FR-011**: Public contracts, internal evidence, highlights and launch commands describe synchronized 1.0.0 and preserve baseline/upstream identities.
- **FR-012**: Publish the implementation PR, resolve every received finding within two rounds and stop at owner final review with green exact-head CI; product tagging/release/deployment remain separately authorized.

### Key Entities *(include if feature involves data)*

- **Candidate**: Six package/source identities, clean revision/version/attempt, documentation and inventory hashes, signing disposition.
- **Promotion election**: Pages or generic destination, expected repository/domain and confined authentication context validated against published authority.
- **Packaged journey receipt**: Extracted runtime/package and elected model identities, transitions, precise accepted authority/recovery comparisons, observed limitations.
- **Launch handoff**: Exact candidate revision/version, measured checks/timings, configuration disposition, short highlights and elected publication commands.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Six real native packages and both exports assemble from one clean candidate through the publication-disabled release path.
- **SC-002**: Full warm PR and release qualification each finish below 600 seconds with their entire successful acceptance set.
- **SC-003**: Invalid promotion inputs fail before writes; valid isolated promotion preserves domain/routes and requires exact build identity.
- **SC-004**: Controlled and elected Windows CPU extracted-package journeys pass precise current-byte/date/model/roster/query recovery comparisons.
- **SC-005**: Green final PR head, all findings answered/resolved, no third review and an honest launch handoff identifying all unexecuted effects.

## Assumptions

- No deployed user audio requires migration; no attended verification gate is introduced.
- Existing Pages serves insonic.io from gh-pages. A new host/domain is out of scope.
- No publisher identities/deployment secret values are configured. Record that state without provisioning credentials or certificate-purchase requirements.
- The approved scope elects controlled and local Windows CPU disposable packaged qualification, not broader acoustic quality claims.
- Owned software/docs/schema stay 1.0.0 because no product release has been published.

## Clarifications

### Session 2026-10-10

- Q: Does implementation PR publication elect a product release/deployment? A: No; dry-run assembly is authorized, owner merge is the stopping boundary.
- Q: Which destination is prepared? A: Existing repository Pages/insonic.io; generic hosting remains supported.
- Q: How should absent signing identities be handled? A: Retain observed unsigned/ad-hoc/unnotarized state and configured-signing behavior without provisioning identities.
- Q: Which packaged acoustic proof is required? A: Existing controlled and elected Windows CPU journeys using matching extracted runtime bytes; other acoustic platforms remain unmeasured.
