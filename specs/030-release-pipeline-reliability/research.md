# Research: Release pipeline reliability

## 2026-10-15: action acquisition boundary

Decision: download approved immutable action archives at step time and invoke
local action paths. Preserve upstream main/post behavior and cache inputs.

Rationale: runner preparation treats 404 as terminal. Local JavaScript actions
register post hooks; both pins declare main/post without unsupported pre hooks.
Historical sources require bootstrap scripts from the defining workflow SHA.

Alternatives: full reruns repeat successful qualification and can mix attempts;
vendoring bundles adds megabytes; rewriting setup/cache logic creates a fork.

- [Runner acquisition](https://github.com/actions/runner/blob/main/src/Runner.Worker/ActionManager.cs)
- [Local main/post handling](https://github.com/actions/runner/blob/main/src/Runner.Worker/ActionRunner.cs)
- [Workflow identity](https://docs.github.com/en/actions/reference/workflows-and-actions/contexts#job-context)
- [Archive guarantees](https://docs.github.com/en/repositories/working-with-files/using-files/downloading-source-code-archives)

Archive compression may change independently of commit contents. Digest changes
require reviewed lock refresh. Lock revisions need explicit maintenance because
Dependabot no longer sees ordinary remote uses.

## 2026-10-15: public verification

Decision: reuse candidate validation, tag and asset metadata verification and the
bounded Pages archive reader. Compare live original bytes and recheck deployment
identity after HTTP requests.

The publisher already hashes all original assets. Rebuilding would invalidate
publication evidence. The prior helper sampled 19 routes and 21 resources; the
archive has 813 files/114 HTML pages. Report an explicit bounded inventory, not
an unrestricted crawl or whole-site visual claim. Latest aliases for historical
only topics deliberately fall back to latest index; use archived bytes.

Both research tasks were delegated as the installed speckit-plan procedure
requires. Agents performed read-only research and confirmed these decisions.
