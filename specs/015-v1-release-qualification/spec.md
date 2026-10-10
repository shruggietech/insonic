# Feature Specification: v1 release candidate and final qualification

**Feature Branch**: `codex/015-v1-release-qualification`

**Created**: 2026-10-10

**Status**: Specified

**Input**: Owner-authorized S015 follows the merged portability slice: integrate the new runtime dependencies, exercise complete fresh-workspace behavior with separate real-engine measurements, prepare synchronized 1.0.0 contracts/documentation, qualify six native package variants and assemble a release candidate. Push and official PR publication are authorized; stop after green exact-head checks and all received review findings are resolved, within two review rounds, for owner merge.

## User Scenarios & Testing

### User Story 1 - Complete a recoverable working library (Priority: P1)

An operator starts an empty workspace, imports media, runs configured processing, associates speakers, elects speaker-model work, queries current evidence and restores the resulting library after source storage becomes unavailable.

**Why this priority**: A release candidate must demonstrate the product journey across shared boundaries, rather than infer completion from independent component fixtures.

**Independent Test**: Automated isolated qualification starts from no application data and checks current authority after processing, queries and recovery. Selected real engines execute separately outside required CI.

**Acceptance Scenarios**:

1. **AC-001**: Given a fresh workspace and committed licensed media, when import, processing, speaker-model preparation/training/matching and querying complete, then results retain canonical audio identity, current Cue JSON, source timing, exact model lineage and queryable accepted evidence.
2. **AC-002**: Given that completed workspace, when a self-contained backup is restored with the original store unavailable, then current recording/transcript/model/roster identities and hashes remain intact and current evidence can be queried without retranscription or training.
3. **AC-003**: Given existing pinned local acoustic models, when explicit maintainer qualification runs on the current candidate code, then its receipt distinguishes generated recognition/diarization/enrollment/matching measurements from deterministic contract checks and lists unexecuted platforms or devices.

### User Story 2 - Obtain an internally consistent v1 candidate (Priority: P1)

An operator receives one exact-source asset set containing CLI-only and desktop packages for every supported platform, matching documentation and schemas, source notices and verifiable inventories.

**Why this priority**: Package/source/version mismatches prevent meaningful release review and reproducible recovery.

**Independent Test**: Candidate collection verifies all six real package receipts and their corresponding asset bytes, exact source, documentation, versions and signing disposition without publishing a product release.

**Acceptance Scenarios**:

1. **AC-004**: Given prepared version 1.0.0, when repository, runtime, documentation and schema validation execute, then all owned version authorities agree while older documented/schema identities and upstream Cueson versions remain unchanged.
2. **AC-005**: Given qualified relocated packages from all three supported native platforms, when the candidate is assembled, then it contains all six variants, matching offline/public documentation, complete corresponding sources and final hashes; missing or mismatched inputs fail.
3. **AC-006**: Given an unconfigured publisher identity or documentation destination, when readiness is reported, then those dispositions are explicit without claiming signing, notarization, product publication or live deployment.

### User Story 3 - Understand and maintain the first major version (Priority: P2)

Operators can understand the documented product contracts, installation prerequisites, supported interfaces and advanced CLI features using matching online/offline documentation.

**Independent Test**: Build and inspect versioned site/offline exports, retained baseline routes, glossary references, capability matrix and concise release highlights.

**Acceptance Scenarios**:

1. **AC-007**: Given all current documentation terms, when the major-version glossary audit completes, then technical terms have precise definitions, primary learning references and local usage links, and public documentation contains no numbered internal planning links.
2. **AC-008**: Given new dependency PR #41, when its exact updates are integrated, then the shared native runtime and credential lifecycle remain qualified on the supported platform matrix.
3. **AC-009**: Given completed implementation, when official PR checks and the two permitted review rounds finish, then every received review comment has a concrete disposition and all actionable threads are resolved before owner handoff.

### Edge Cases

- Old schema/documentation routes must not be rewritten to 1.0.0 identities.
- Package receipts from earlier commits, mismatched variants or different source recipes cannot qualify the candidate.
- A development checkout is not an officially published product, even after owned versions become 1.0.0.
- Real-engine absence, inaccurate predictions and device/provider failure require measured or explicit failure results, not invented quality claims or silent fallback.
- Restore must exclude source credentials and local tool paths while preserving accepted portable authority.
- Cold source compilation and full cached CI timing must be reported separately; the prior slice's specific cold exception is not a blanket waiver.

## Requirements

### Functional Requirements

- **FR-001**: Qualification MUST exercise fresh-workspace import, processing, speaker-model preparation/training/matching, queries and self-contained backup/restore as one connected journey.
- **FR-002**: Qualification MUST verify exact current audio/transcript/model lineage, source timing, roster and query authority after restoration with the source store unavailable.
- **FR-003**: Explicit real-engine qualification MUST record current code/tool/model identities, settings, device, resource/runtime bounds, generated outputs and acoustic diagnostics independently of required CI.
- **FR-004**: Required CI MUST retain its no-acoustic-inference/no-model-download contract; fixture output MUST NOT be described as measured model accuracy.
- **FR-005**: Owned software, runtime, documentation, master schema and package metadata MUST prepare version 1.0.0 together, retaining baseline routes and immutable old identities.
- **FR-006**: The candidate MUST contain both package variants for Windows x86-64, Linux x86-64 and macOS ARM64 plus matching documentation and source-complete inventories from one clean exact revision.
- **FR-007**: Candidate validation MUST reject missing variants, stale source/version identities, corrupted bytes and incomplete source closure.
- **FR-008**: Release preparation MUST refresh the major-version glossary and concise highlights/changelog, document actual capability gaps and avoid public planning/evidence links.
- **FR-009**: New dependency updates in PR #41 MUST integrate with current pins and preserve native runtime and credential acceptance.
- **FR-010**: Readiness MUST report configured signing/deployment status without selecting new destinations, provisioning identities, tagging, releasing or deploying during this slice.
- **FR-011**: Implementation MUST complete Spec Kit analysis/convergence and required checks; cached full CI MUST meet the ten-minute target or record a concrete unresolved deviation rather than claim compliance.
- **FR-012**: Official PR publication MUST include issue mapping, exact-head results and all received review dispositions; no third review round or owner merge may be performed.

### Key Entities

- **Journey receipt**: Exact code and fixture/tool identities; operation transitions; accepted current identity/hash comparisons; recovery and query results.
- **Acoustic qualification receipt**: Selected engines/models/device, actual generated diagnostics, bounded runtime/resources and explicit unexecuted claims.
- **Release candidate**: One clean source revision, synchronized version, six package identities, documentation archives, source/inventory hashes and signing disposition.
- **Readiness record**: Implemented/tested behavior, external configuration status and publication/deployment operations not performed.

## Success Criteria

### Measurable Outcomes

- **SC-001**: One connected fresh-workspace journey passes through all elected operations and recovery, preserving every compared current authority identity and byte digest.
- **SC-002**: Current elected local engines execute outside CI with retained measured results; recognition, diarization and speaker-model diagnostics are separately described without unsupported accuracy claims.
- **SC-003**: All six native package variants pass relocation acceptance and assemble into one validated exact-source 1.0.0 candidate with matching documentation.
- **SC-004**: All owned version authorities agree on 1.0.0; baseline documentation/schema identities remain accessible and unchanged.
- **SC-005**: All required repository/product/site checks pass on the final PR head, and every received review finding is addressed within the two-round cap.

## Assumptions

- No deployed user audio requires bulk migration. Existing fixture/scratch workspaces are disposable only within explicitly owned qualification directories.
- The owner selected 1.0.0 preparation through the approved next-slice approach; this is not authorization for a tag, public product release or documentation deployment.
- Advanced offline backup and low-level storage/profile operations may remain CLI-first with an explicit capability matrix; no unrelated GUI expansion is required.
- Existing local acoustic environments/model bytes are preferred; no unselected hosted provider or new external routing is introduced.
- All existing storage/catalog/graph alternatives remain fully supported and their meaningful acceptance checks remain.

## Clarifications

### Session 2026-10-10

- Candidate versus publication: prepare 1.0.0 locally and qualify its exact-source assets; retain explicit unpublished status until a separately authorized tag/release. This matches the owner-approved next-slice proposal.
- Acoustic scope: execute current elected CPU engines with existing pinned local models and report recognition, diarization and enrollment/matching separately; all required CI remains deterministic for acoustic operations. Unexecuted device/platform quality is explicit, not generalized from Windows.
- Interface parity: retain approved advanced CLI-first backup/profile tooling with documented lag instead of adding unrelated GUI work.
- Historical compatibility: preserve immutable baseline schemas and old routes; do not blindly replace historical fixture identities when current bindings change.
- Review limit: automatic initial reviews plus at most one explicitly requested second round, including any companion security request on that same head. All comments received in those rounds require disposition.
