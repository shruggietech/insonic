# S078 Workflowy maintainer integration

**Feature Branch** `codex/078-workflowy-maintainer-integration`
**Created** 2026-11-20
**Status** Implemented and locally verified

## User Scenarios & Testing

### User Story 1 - Find the project notebook (Priority: P1)

A maintainer or authorized agent begins substantive insonic work with the standard
Workflowy instructions available from the checkout. They can find the existing
project, read the current notes and record a handoff in the right place.

**Independent test** Compare the bundled guidance with its selected source.
Follow its documented lookup through an available Desktop MCP connection and
confirm that the workspace instructions and the existing project can be read.

1. **Given** a checkout with the integration, **when** a maintainer reads the
   active repository instructions, **then** they find one canonical Workflowy opt-in.
2. **Given** an available skill loader, **when** it reads the bundled skill,
   **then** both companion guides resolve from the skill directory.
3. **Given** a connected Desktop MCP, **when** the agent follows the lookup,
   **then** it reaches the existing insonic branch under the instructed workspace.

### User Story 2 - Continue when the notebook is unavailable (Priority: P2)

A maintainer can complete repository work even when the optional local
connection is unavailable. The handoff remains useful and official progress
continues to come from the established tracker.

**Independent test** Inspect the opt-in and guides for the unavailable-connection
handoff and compare the changed-file set for the privacy and product boundaries.

1. **Given** an unavailable skill loader or Desktop MCP, **when** the agent
   begins substantive work, **then** the repository instructions retain the
   working procedure and direct it to continue with an appropriate handoff.
2. **Given** a repository change prepared for review, **when** its contents are
   inspected, **then** it contains general maintainer guidance and the approved
   skill, while notebook contents, private node links and connection material
   stay in their existing locations.

### Edge Cases

- An older continuity section exists. Replace it in place and preserve all
  surrounding repository instructions.
- The project already exists in the notebook. Reuse its branch and read it back.
- A connection is unavailable or returns ambiguous results. Follow the documented
  guidance for scoped lookup, uncertainty and continuing the work.

## Requirements

### Functional Requirements

- **FR-001** Provide the owner-approved, versioned Workflowy skill with both
  self-contained companion guides in the repository's agent skill location.
- **FR-002** Replace the earlier project continuity block with exactly one
  current opt-in from the selected skill, preserving the surrounding instructions.
- **FR-003** Keep the workflow usable from the repository instructions if the
  skill loader or Desktop MCP is unavailable. With a working connection, guide
  agents to check workspace instructions, reuse the established project branch
  and read back changes.
- **FR-004** Keep private notebook history, private node links, local connection
  material and workstation-specific details out of repository artifacts.
- **FR-005** Preserve product, installation, versioned documentation, schema,
  build and release behavior. Keep this work scoped to maintainer guidance and
  its project provenance, planning and changelog records.
- **FR-006** Record local checks and their limits. Keep GitHub authoritative for
  published tracking and defer publication to the autopilot pre-push handoff.

## Success Criteria

### Measurable Outcomes

- **SC-001** A clean checkout contains one opt-in and all three approved skill
  files. Both guide references resolve with no local-profile dependencies.
- **SC-002** One connected session confirms the workspace instructions and the
  existing project branch. The written fallback covers an unavailable connection.
- **SC-003** Review of every changed file finds zero private notebook records,
  node links, connection credentials or out-of-scope product changes.
- **SC-004** Repository and affected documentation checks pass locally and the
  recorded handoff identifies any verification that remains pending.

## Assumptions

- The owner approved the maintainer-only scope. The private backfill was a
  separate notebook operation, and desktop product work is a separate slice.
- A live lookup depends on the workstation's existing Desktop MCP connection.
  The integration does not need machine configuration or a product dependency.
- The implementation target is a local feature commit pending push
  authorization. A tracking brief supports the later GitHub handoff.
