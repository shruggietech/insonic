# Development workflow

## Tracking and scope

GitHub Issues are the primary work intake, milestones define release outcomes and a GitHub Project provides status. Issues state concrete acceptance, affected versions for defects, dependencies and scope exclusions. The board tracks issues and pull requests without duplicating task text. Use statuses Backlog, Ready, In progress, In review and Done, with Slice, Release, Priority and Area fields.

[Delivery outcomes](roadmap.md) describe product dependencies. Numbered implementation plans and verification records live under `specs/`; public documentation describes system contracts and does not render those planning artifacts.

## Spec Kit slices

Spec Kit organizes bounded, observable work. Python 3.11 or newer supports its cross-platform maintainer helpers. The application runtime uses the language and dependency boundaries defined in [technology decisions](technology.md).

```mermaid
flowchart TB
  Issues[Related GitHub issues and release milestone] --> Specify[Specify observable outcome]
  Specify --> Clarify[Clarify unresolved requirements]
  Clarify --> Plan[Plan contracts, data and implementation]
  Plan --> Tasks[Order tasks and acceptance checks]
  Tasks --> Analyze[Analyze consistency and scope]
  Analyze --> Implement[Implement and run focused checks]
  Implement --> Converge[Converge against acceptance]
  Converge --> PR[Pull request with evidence]
  PR --> Merge[Human-approved squash merge]
  Merge --> Release[Release with matching docs and schema]
  Release --> Observe[User observations filed as versioned issues]
  Observe --> Issues
```

A slice has `spec.md`, `plan.md` and `tasks.md`, with research, data-model, contracts and checklists where useful. Small work stays small; empty artifacts are unnecessary. Resolve critical consistency findings before implementation. Keep each branch and pull request focused on a coherent outcome. Dates and internal progress entries are chronological.

Resolve routine implementation choices against approved product requirements. Clarify unresolved decisions that materially change behavior.

## Automated checks and user observations

Before merging, run checks relevant to changed behavior, dependency/artifact integrity and reproducible builds. Secret redaction, archive extraction, job attempt ownership and graph publication require meaningful tests. Reversible text changes need formatting and link checks.

Users ordinarily verify released behavior and report observations through issues. A report includes version, platform, expected/actual behavior and a minimal reproduction. Correlate symptoms before asserting a cause. A green unrelated unit suite does not resolve a reported field observation.

## CI budget

Repository CI has parallel `foundation` and `docs` jobs. The first checks repository integrity and maintainer tests; the second builds and checks the static documentation export. Each has a ten-minute timeout. Cache dependencies, cancel obsolete branch runs and consume committed assets and lockfiles. Network brand freshness checks are manually triggered.

Measure end-to-end turnaround. When it approaches ten minutes, remove redundant work, tighten change scopes or parallelize independent checks. Native package checks use a relevant platform matrix, with expensive packaging in release jobs. Ordinary pull-request checks do not invoke live models or download large model weights.

## Documentation and contract governance

Public documentation states the authoritative system behavior. Keep slice identifiers, task records and verification evidence in internal planning. Historical and version-change descriptions belong in the [changelog](changelog.md). The documentation index carries the concise implementation-status statement.

Use numbered steps for a procedure with one linear path. Use a diagram when branches, relationships, dependencies, feedback or retry paths explain the system more clearly. Keep Mermaid flowcharts top-down with `TB` or `TD` orientation.

The software, documentation and master JSON Schema share one release version. The [JSON contracts](contracts.md) define machine-readable boundaries; the [logical catalog schema](schema.md) defines relationships and integrity. Prefer backward and forward compatibility where practical. Explain breaking changes in the changelog and concise release highlights, including the migration or affected interface.

For every major release, rescan all documentation for technical terms and refresh the [glossary](glossary.md). Each entry explains the exact term, links a primary external learning reference and points to the local pages that use it most heavily.

## GitHub setup and approval

The maintainer bootstrap configures Issues, squash-only merges, automatic merged-branch deletion, labels, milestones and a Project. Branch protection requires the `foundation` and `docs` checks and resolved conversations. Activate protection after those checks exist.

`npm run github:check` reads setup status. `npm run github:apply` applies prepared settings to an existing repository with repository-administration and organization-Project permissions. It does not create or push a repository. Repeated setup reuses tracked issues, milestones and the Project, preserving edits, existing values and archived work. Initial status setup applies only to a new empty Project. Missing required status options or established-item metadata are reported for completion. After a partial failure, resolve permissions, rerun and confirm settings and membership through API readback.

One human-approved squash merge is the standard approval. A sole owner cannot approve their own pull request, so administrator authority and the manual owner merge remain explicit. Automated analysis and focused checks precede that merge.

See [contribution guidance](../../CONTRIBUTING.md) and [release workflow](releases.md).
