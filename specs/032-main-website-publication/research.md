# Research: S032

Research completed 2026-10-15 using Spec Kit research agents and direct code/run inspection.

## Publication and transport

**Decision**: Separate website.py and manual workflow. Validate existing release read-only, keep site_revision and release_revision distinct, require newest exact push/main CI and recheck current main. Use isolated Git, credential environment, non-force parent fence and exact lost-response reconciliation. Replay qualified archives and preserve old schema bytes.

**Rationale**: Original release archives correctly bind product tag/source and cannot publish later corrections. Prior approximately 43 MB inline tree API request returned 422 (cause unproven); Git published the same export successfully. Next.js rebuilds can change build IDs.

**Alternatives**: Weaken original release checks, replace release archives, force push, large inline API trees or one request per blob (rejected).

## Live evidence

**Decision**: Fetch bounded fresh production HTTPS responses and render those verified bytes in an owned headless browser. Four widths, both themes, shared geometry/readability checks. Check five package HEAD statuses/sizes through approved CDN redirects; do not log signed URLs. Clearly label bounded live-response rendering.

**Rationale**: Fixtures alone cannot prove deployment; unrestricted browser continuation cannot bound payloads. Binary-byte verification stays in original release tooling.

**Alternatives**: Personal-browser automation, fixture-only proof and downloading 650 MB for website-only changes (rejected).

## CI

**Decision**: Overlap scope/foundation; narrowly classify known site files; split docs-build from stable docs qualification; share source/run/attempt/hash-bound exports with no fallback. Retain dependency installation for notices, all native/MSI checks, candidate docs gate and independent release builds.

**Rationale**: Run 38752604343 took 671 seconds overall/667 active. Foundation, Windows source, desktop package and aggregation formed the critical path. Repeated help compilation costs about 25–30 seconds; scope overlap approximately 10–12. Combined prediction must be measured. Native assets/package assembly require offline tree only.

**Alternatives**: Skip native tests, enlarge limits or rely on scope overlap alone (rejected).

## Resolution

No NEEDS CLARIFICATION remains. Final hosted timing, with queue/active boundaries, decides turnaround acceptance.
