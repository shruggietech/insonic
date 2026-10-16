# S033 verification

## Scope and authority

S033 implements #140, #141, #142, #143 and #145 on `codex/033-media-lifecycle-correctness`, based on `ea8537e38beba6d9724df1a9ec1d87013eefbd6b`. The owner authorized push, a PR and up to two review rounds. Merge and release remain owner actions. #114 is only partially advanced; #130 and #147 remain open.

## Local implementation verification (2026-10-15)

All commands ran through hidden noninteractive children. Native tests used the prepared exact media tools and native library environment. Results describe the S033 working tree; final committed-head CI is recorded separately.

| Check | Result | Evidence |
| --- | --- | --- |
| Regression-first lifecycle cases | Expected failures observed before fixes | Held report lease/retry, cleanup failure precedence, canonical MIME, manifest-only parser, real SubRip extraction and timeout/limit/cancel classification |
| Focused lifecycle and recovery tests | Passed | Lease release/expiry, accepted retry, real delete failure, runtime startup sweep, durable transcript causes and prior-document preservation |
| `go test ./...` | Passed | `build/s033-full-go.log` |
| `go test -tags system_ladybug ./...` | Passed | `build/s033-system-ladybug-go-rerun.log` |
| `go vet ./...` | Passed | `build/s033-go-vet.log` |
| `npm run check` | Passed | `build/s033-npm-check.log` |
| `npm test` | 118 passed | `build/s033-npm-test.log` |
| Python `unittest discover`, `*_test.py` | 300 passed | `build/s033-python.log` |
| Site check and tests | Passed; 15 tests | `build/s033-site-check.log`, `build/s033-site-test.log` |
| Site public/offline build | Passed | `build/s033-site-build.log` |
| Relocated Windows CLI package | Passed (227.484 seconds) | `build/s033-package.log`; all five lifecycle cases passed in 46.031 seconds, plus existing documented workflows |
| UTF-8, LF, mojibake and whitespace | Passed | Source, tests, docs and internal planning; `git diff --check` |

The first full native run encountered the existing 10 ms assistance cancellation test returning `unavailable` during concurrent source compilation. Its isolated rerun and a subsequent full native run passed; no test or timeout was weakened. An earlier `-tags native` run is ordinary-test evidence only; the actual native adapter uses `system_ladybug`.

Local package preparation initially found stale companions and a combined source-build receipt without the CI dependency-group provenance. Preparation then passed through the existing independent compression/audio/AV1 build and verified merge path. No package integrity gate has been bypassed. The local CLI archive SHA-256 is `0bec943a10c999c8e1893e70c8d2685de391cab561597b069c55e02202ea6c98`; its receipt truthfully records the base revision and `source_dirty: true`. It is qualification evidence, not a published release candidate. Fixture expectations were corrected to distinguish canonical mono MP3 facts from the WAV preview actually returned for AAC input.

## Convergence

Cross-artifact analysis mapped all 12 functional requirements. Implementation convergence added T021 for lease expiry and T022 for parsed cue text/exact start-end assertions; their focused regressions passed. The five installed lifecycle cases also passed in a targeted built-CLI run (26.796 seconds). Existing durable cleanup rows and startup/periodic sweeps provide recovery; no new worker or catalog migration was introduced. Fixed diagnostic notices reuse the existing result schema rather than changing an immutable published schema.

## Publication and review

Local package qualification passed. Commit and PR publication are next. Five issues are assigned to milestone 17 and Project 7, slice S033, status In progress. Opening the PR starts review round 1; at most one further round is authorized. Final-head checks and unresolved-thread counts will be verified directly before handoff.

## Limits of evidence

Local native runtime and package work target Windows x86-64. Hosted CI must verify Linux, desktop packaging and alternate PostgreSQL/S3 backends. This slice runs deterministic media/loopback fixtures without inference, model downloads or owner media. It does not complete the broad fixture corpus in #114 or all acquisition diagnostics in #130. No release or owner-facing installation has been performed.
