# S078 implementation plan

**Branch** `codex/078-workflowy-maintainer-integration`
**Date** 2026-11-20
**Specification** [spec.md](spec.md)

## Summary

Install the pinned Workflowy skill and switch agent instructions to the
canonical project opt-in. This is a maintainer workflow change. Preserve the
private notebook and application surfaces.

## Technical context

| Concern | Choice |
| --- | --- |
| Content | UTF-8 Markdown with LF line endings |
| Skill source | Workflowy v1.0.1, pinned upstream and byte-verified |
| Discovery | Repository skill directory and project-level `AGENTS.md` |
| Connection | Existing operator-configured Desktop MCP |
| Verification | Blob/link comparison, scoped MCP readback, root and site checks |
| Tracking | Local feature commit and GitHub-ready brief, with publication at handoff |

## Constitution check

| Principle | Application |
| --- | --- |
| I | Implement the approved maintainer-only scope. Desktop work is a separate slice. |
| II | CLI and desktop contracts stay outside the changed-file set. |
| III | No product source, data, corpus, or media changes enter this slice. |
| IV | Connection material stays in existing workstation configuration. |
| V | Use Spec Kit, maintain verification boundaries, and stop at the pre-push handoff. |

The design requires no added runtime library, schema, deployment, or CI changes.

## Project structure

- `.agents/skills/workflowy/` contains three upstream files.
- `AGENTS.md` adopts the skill's current block in place.
- `NOTICE` retains relevant upstream attribution.
- `CHANGELOG.md` records the maintainer change under Unreleased.
- This slice directory contains the specification, verification steps and
  implementation evidence.

## Implementation order

1. Validate the specification and the referenced upstream source.
2. Install the skill and replace the previous opt-in, then record attribution
   and the unreleased change.
3. Verify source parity, file boundaries, the connected lookup and the affected
   repository checks.
4. Record convergence and commit the reviewed changes locally.

See [research](research.md) for the decisions and [quickstart](quickstart.md) for
the acceptance checks. The [data inventory](data-model.md) identifies which
content belongs outside the repository.

## Tracking brief

**Title** S078 Workflowy maintainer integration

Install Workflowy v1.0.1 and the canonical repository opt-in so maintainers can
find and maintain the existing notebook from project instructions. Accept the
change when the three vendored files match upstream, both local references
resolve, the opt-in occurs once, the connected lookup reaches the existing
project, and root/site checks pass. Git records contain general maintainer
guidance and implementation evidence. The private backfill stays in Workflowy.

Use a focused maintainer issue and the
[insonic GitHub Project](https://github.com/orgs/shruggietech/projects/7) for
published progress. Link the issue and a maintainer milestone in the later PR.
