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

## Cold Go dependency preparation (2026-10-20)

Decision: prepare external and standard Go libraries under the exact CLI, launcher and desktop contexts before readiness, with an empty isolated cache and a same-attempt verified transfer. Independent planning research confirmed Go list without -export discovers the graph without compiling product packages. Main-module packages and executable dependencies are forbidden build targets.

Rationale: first hosted assembly had a Go cache miss and spent 206 seconds building, pushing interactive completion beyond 660 seconds and archive into its job timeout. Existing optional cache reuse cannot guarantee the cold case. CGO flags differ between package variants, so both native paths must be represented. Go cache does not detect changed external C libraries, making native/compiler identity necessary.

Alternatives: a warmed retry provides comparison only; moving application builds into preparation violates the timing contract; increasing native or job deadlines weakens the intended acceptance. Sources: [Go build caching](https://pkg.go.dev/cmd/go#hdr-Build_and_test_caching), [Go package listing](https://pkg.go.dev/cmd/go#hdr-List_packages_or_modules).
