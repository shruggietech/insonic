# S078 research and decisions

## D1. Source and layout

**Decision** Vendor Workflowy skill v1.0.1 from the [upstream introduction
commit](https://github.com/shruggietech/skills/tree/15c215ac8ed42688e92828191a0cccc38dfec8d0/skills/workflowy).
Keep its `assets/` layout and retain attribution in the root notice.

**Rationale** The three blobs also match the published
[skills v1.12.0](https://github.com/shruggietech/skills/releases/tag/v1.12.0)
package and [Fragcap's opt-in](https://github.com/shruggietech/fragcap/commit/ef2b5626).
The pinned files resolve their own references consistently across checkouts.

| File under the skill directory | Upstream Git blob |
| --- | --- |
| `SKILL.md` | `c874e347ba54a5cf61396ded02bc60174db78df4` |
| `assets/project-backfill.md` | `d0d98d8ae03f04f4355fe72c7de051fde7bf4284` |
| `assets/workflowy-mcp.md` | `37a3d27f9ba1f96a5ba47ca8a31fc963772536a8` |

**Alternatives considered** Relying on a profile-installed skill leaves fresh
checkouts dependent on machine state.
[Cueson's opt-in](https://github.com/shruggietech/cueson/commit/6790b9f) and
[Go Schedule's opt-in](https://github.com/shruggietech/go-schedule/pull/491)
use a sibling distribution of the same guides under `references/`. Using the
canonical upstream layout preserves all three source blobs. Insonic has no
tracked skill lockfile or separate Claude project instruction file to update.

## D2. Opt-in and privacy boundary

**Decision** Replace the previous opt-in in place with the fenced block from
the pinned skill. The connected session uses the workstation's existing Desktop
MCP settings. Keep the existing project and its contents in Workflowy.

**Rationale** One source block defines a coherent lookup and handoff path while
preserving the surrounding repo instructions. The repository can describe
lookup and fallback without concrete notebook or authentication data.

**Alternatives considered** Keeping both sections produces conflicting guidance
about project location. A hard-coded node link couples public source to a
private workspace and publication audience.

## D3. Verification scope

**Decision** Run the normal root checks and the affected documentation tests
and builds. Verify upstream blob identity, internal links, text integrity,
privacy boundaries and connected lookup explicitly.

**Rationale** The root checker excludes `.agents/`. The site renders the root
changelog in the specification baseline, so the changelog change has a site
surface. The current hosted scope classifier elects its full suite when a new
`.agents/` file appears. Its result is a later hosted check.

**Alternatives considered** Root checks alone miss vendored skill integrity.
Altering the classifier or adding an application-level service requirement
would broaden the slice.
