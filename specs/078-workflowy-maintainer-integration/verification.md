# S078 verification

## Local result

The checkout includes the pinned Workflowy v1.0.1 skill, both companion
guides and one canonical project opt-in. The previous opt-in was replaced
in place. All surrounding instructions were preserved. The root notice
retains the upstream attribution, and Unreleased records the maintainer
change.

## Analysis and convergence

The read-only prerequisite check resolved this slice before implementation.
Analysis traced two stories, six requirements, four outcomes and the five
constitution principles to the plan and tasks. No blocking conflict remained.
After implementation, the same inventory was checked against the changed files
and local evidence.

| Requirement | Evidence |
| --- | --- |
| FR-001 | All three installed Git blobs match the pinned [source](research.md). Both guide targets resolve. |
| FR-002 | The installed opt-in matches the upstream fenced block. The old heading occurs zero times and the new heading once. The original outer text matches byte for byte. |
| FR-003 | Desktop MCP reached ABOUT, PROJECTS and the existing insonic branch. The written unavailable-connection path directs the agent to continue and leave an appropriate handoff. |
| FR-004 | The changed-file review checked for private notebook records, private node URLs, connection material and machine paths. |
| FR-005 | The diff is confined to maintainer instructions, upstream skill assets, attribution, the changelog and slice artifacts. Both static exports render the changelog entry. |
| FR-006 | Local checks and their limits are recorded here. The tracking brief and later publication task are in the [plan](plan.md) and [tasks](tasks.md). |

## Executed checks

| Check | Result |
| --- | --- |
| `npm run check` | Passed foundation, schema and contract validation. |
| `npm test` | 223 passed, zero failed or skipped. |
| `npm --prefix site test` | 18 passed, zero failed or skipped. |
| `npm --prefix site run build` | KaTeX regression passed. Web and offline exports passed integrity checks across 289 pages each. |
| Changed-file review | Pinned blob identity, exact opt-in, link resolution, UTF-8 without BOM, LF, mojibake and Git whitespace checks passed. |
| Desktop MCP | Scoped read verified the existing workspace and project from the connected session. |

Local preparation checked Node v26.5.0, npm 11.17.0 and 12 direct
dependencies against their installed lockfile versions. Preparation took
0.33 seconds. Local command qualification through the terminal site build
took 41.35 seconds, including the gap between the parallel tests and the
build, for 41.68 seconds combined. The final evidence and whitespace
review followed that campaign.

## Remaining verification

Skill discovery was checked from the repository location and front matter.
A fresh agent process and a deliberately disconnected Desktop MCP session
were outside this local run. The custom requirements checklist retains its
reviewer-owned markers. Hosted exact-head CI and GitHub tracking follow
publication authorization.
