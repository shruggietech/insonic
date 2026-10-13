# Feature Specification: Package dependency integrity

**Feature Branch**: `codex/024-package-dependency-integrity`

**Created**: 2026-10-13

**Status**: Specified

**Tracking**: [#88](https://github.com/shruggietech/insonic/issues/88), milestone Post-release usability, Project 7.

**Input**: S024 executes the approved objectives: fix #86, consolidate PR #87 and remediate the Mermaid/KaTeX audit findings. Automatic push and official PR are authorized. Resolve every finding in at most two review rounds, verify final-head green CI and stop for owner final review and squash merge.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Receive truthful shipped dependency notices (Priority: P1)

A recipient can identify the actual dependencies included in their selected CLI or desktop package and read the matching upstream notices.

**Why this priority**: Missing source directories currently permit project notices to be attributed to unrelated modules.

**Independent Test**: Compare linked dependency identities with the package inventory and notice bytes; incomplete source evidence refuses assembly.

**Acceptance Scenarios**:

1. **Given** a CLI or desktop package, **When** notices are assembled, **Then** the runtime dependency set matches the union shipped by that variant and every entry has matching version and upstream bytes.
2. **Given** absent, empty, invalid or mismatched source metadata, **When** assembly resolves notices, **Then** it fails without substituting project notices or silently omitting a shipped dependency.
3. **Given** unused tooling modules in the selected graph, **When** packaging runs, **Then** they neither acquire false shipped identities nor require unrelated downloads merely to populate notices.

### User Story 2 - Retain behavior with elected dependency fixes (Priority: P1)

Users retain Windows/Linux CLI, desktop and six-backend operation with the elected runtime update, while documentation uses a patched diagram dependency closure.

**Why this priority**: These dependencies reach the runtime and every package's help.

**Independent Test**: Qualify exact elected versions, locked closure and the targeted advisory.

**Acceptance Scenarios**:

1. **Given** captured PR #87, **When** consolidated, **Then** x/net 0.61.0 and its verified closure are synchronized without changing elected Go/native toolchains, routes or credential contracts.
2. **Given** Mermaid 12.2.0 and patched KaTeX 0.18.2, **When** the locked site tree is installed, **Then** GHSA-238p-pmpm-9mq7 entries are absent and package notices match the new closure.

### User Story 3 - Keep portable documentation and delivery evidence (Priority: P2)

Readers retain complete public/offline documentation, local diagrams and historical routes; maintainers receive one qualified consolidation.

**Why this priority**: The diagram closure ships in both variants and must retain the delivery boundary.

**Independent Test**: Qualify public/offline and release-target documentation, retained historical routes and every package from one source and attempt.

**Acceptance Scenarios**:

1. **Given** the updated renderer, **When** exports are produced, **Then** diagram/theme behavior, local resources, pages, schemas and historical URLs remain usable and frozen 1.0.0 source has zero changes.
2. **Given** the official S024 PR, **When** reviews and qualification finish, **Then** every finding is answered/resolved within two rounds and the final head has green checks before owner handoff.

### Edge Cases

Absent/empty source directories; modules without notices; version mismatches; replacements; shared dependencies; CLI-only omission of desktop dependencies; duplicate/conflicting identities; unused tooling; historical inventories; diagram closure drift; prototype-polluted options; corrupt/mismatched source/attempt artifacts; later bot rebases; delayed reviews; hidden Windows children.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Current runtime notice inventories MUST represent the union actually shipped by the elected variant, including exact module identities/versions, matching upstream notices and the elected compiler notice.
- **FR-002**: Missing, empty, invalid or identity-mismatched source metadata MUST fail assembly. Project/current-directory notices MUST NOT substitute for upstream bytes; missing notices/conflicting identities MUST NOT be silently accepted.
- **FR-003**: Unused tooling MUST NOT be recorded as shipped runtime modules or trigger unrelated full-graph downloads for accounting. Replacement identities and dependencies shared by executables MUST resolve consistently and completely.
- **FR-004**: Include captured PR #87's x/net 0.61.0, Mermaid 12.2.0 and locked patched KaTeX 0.18.2. Moving heads or later latest versions MUST NOT silently change the election.
- **FR-005**: Targeted GHSA-238p-pmpm-9mq7 audit entries MUST be absent after clean installation; patched renderer-option behavior MUST have meaningful regression coverage. Introduce no unrelated product eligibility/moderation policy.
- **FR-006**: Retain qualified CLI/desktop behavior, all six backends, protected credentials, elected routes, cancellation, ownership, source completeness and signing disposition.
- **FR-007**: Retain complete public/offline pages/resources, local diagrams/theme behavior, historical URLs and truthful candidate/release rendering. Frozen 1.0.0 docs/schemas MUST remain unchanged.
- **FR-008**: Software/docs/master-schema stay unreleased 1.0.1 candidate. Existing package record shapes and historical validation remain readable; accurate current membership may change without rewriting historical records.
- **FR-009**: Required repository/runtime/frontend/site/source/cache/live-backend/native/four-package/candidate checks MUST pass, retaining integrity checks, hidden noninteractive children, ten-minute job bounds and no ordinary-CI inference. Qualification binds one clean source and attempt.
- **FR-010**: Use linked issue/milestone/Project tracking and one official PR. Publish under standing authorization, answer/resolve every finding within two rounds, retain PR #87 until owner merge and stop without product release/tag/deployment/merge.

### Key Entities

- **Shipped dependency identity**: Runtime membership, version and matching upstream notice references for a selected package.
- **Dependency election**: Source PR/advisory, captured commit/package identity, exact version and locked closure.
- **Qualification evidence**: Clean source/attempt, platform/variant, outcome, retained artifact identities and observed limits.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every shipped runtime dependency has matching version/notice evidence, with zero project substitutions and zero unrelated tooling entries in all four variants.
- **SC-002**: All three elected fixes are traceable and the targeted advisory has zero audit entries after clean installation.
- **SC-003**: Every expected public/offline page/resource and historical route passes qualification, with zero frozen source changes.
- **SC-004**: Full final-head qualification passes, every review thread is resolved within two rounds and the checkout is clean at handoff.

## Assumptions

- The owner restated S024 to execute the prior recommendation; #77-#83 belong to merged S023 and are not reopened.
- Package accounting and elected fixes are in scope; managed workers, first-run setup, MSI, release and application contract redesign are separate.
- Native/compiler pins remain elected absent a demonstrated compatibility necessity. Routine interface/accounting decisions are resolved under autopilot.
- Targeted remediation does not assert that every possible vulnerability or unexecuted behavior is absent.

## Clarifications

### Session 2026-10-13

- Q: Which scope applies? A: The owner explicitly restated S024 to execute #86, PR #87 and the Mermaid/KaTeX fixes; completed #77-#83 stay with S023.
- Q: Which versions are elected? A: Captured PR #87 head 576ef85aacbf1bcac1682ec898e56f29624d9869, x/net 0.61.0, Mermaid 12.2.0 and patched KaTeX 0.18.2; no later versions inferred.
- Q: Is this a release or new notice wire contract? A: No. Preserve current record shape and historical readability, qualify one consolidation and stop for owner merge.

Coverage: scope/data/identity/failure/recovery/security/performance/external dependencies/terminology/completion are Clear. Application UI/localization and model inference are intentionally outside this maintainer slice. No critical ambiguity requires human input; 16/16 checklist items remain passing.
