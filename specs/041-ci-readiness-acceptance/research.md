# Research: CI readiness and acceptance

## Decision: actual readiness owns consumption

The checked-out package-windows job needs only scope and starts its bounded transfer wait before the source producer. Its successful cold producer can exceed 180 seconds. Keep the helper bound; gate consumers on verified terminal readiness. Retain exact cache and source/recipe/compiler validation. Source inspection and independent Spec Kit research review agree.

GitHub documents prerequisite success and failure/skip propagation in [job dependencies](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-jobs) and PR cache restrictions/eviction in [cache reference](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching). No sibling PR cache is assumed available on main.

## Decision: separately measured phases

The owner approved separate preparation and 660-second qualification, reporting both and total. All qualification starts after the readiness boundary. This avoids excluding checks that ran during preparation. A 15-second aggregate finalization allowance is explicit and requires later terminal observation, not a promise about provider latency.

## Decision: use existing producer trust

No cross-run source import. Reusable ready validates probe success plus either verified warm upload or successful cold ffmpeg upload. Exact current-run transfer restore and independent candidate validation continue. Package inventories record resolved versions; they do not strengthen the compiler identity into an unsupported fully pinned toolchain claim.

## Decision: aggregate is always evaluated

Require explicit success for elected work, skipped for deliberate non-election, and failure on unknown/missing results. Documentation-only and maintenance live-site elections remain explicit. Query the exact run attempt and complete bounded job inventory for timing; reject ambiguous identities. GitHub [job](https://docs.github.com/en/rest/actions/workflow-jobs) and [run](https://docs.github.com/en/rest/actions/workflow-runs) APIs are read-only.
