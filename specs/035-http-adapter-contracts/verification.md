# S035 verification

## Analysis gate

Read-only analysis found coverage for all seven functional requirements and five success criteria across twelve tasks. No critical or high findings remain. Constitution gates pass. Six contract checklist questions remain reviewer-owned. Explicit owner autopilot authorization permits implementation to proceed.

## Execution

Regression tests precede product changes. Evidence follows in chronological order.

## Local implementation and checks (2026-10-17)

- Regression-first checks exposed legacy extraction alias rejection, active-schema reason omissions and absent operation-specific limit context. The first full matrix also exposed an unbounded test-server teardown; the fixture now has its own finite wait.
- Targeted native regressions pass for both adapters across authentication, missing models/routes, busy services, redirects, malformed/duplicate/trailing/truncated responses, body failures, byte caps, transport refusal, deadlines and explicit cancellation. Invalid proposal content is reclassified at the provider boundary. App tests cover late-success timeout, malformed fixture proposals and proposal byte ceilings.
- Full system_ladybug Go suite passed (78.484 seconds); vet passed (15.219 seconds). Root check and all 118 Node tests, all 304 Python tests, frontend type/behavior checks, and site check/tests/public+offline build passed. Final affected-check repetition follows fixture completion.
- Fresh Windows CLI package built successfully. Exact installed qualification passed in 10.156 seconds: 130 cues/two chunks, both public aliases, canonical provenance, unknown/current-other-chunk/current-foreign-media citation rejection, unknown/clock-spoof rejection, authoritative source clocks, transcript-search availability, nine provider failure cases, native 501 rows, artifact 25 bytes/max 24, provider response 1025 bytes/max 1024, and human materialization correction. Exact document pages preserve integers and bind the document digest.
- Fixture-only corrections used the existing document byte continuation and unavailable exit 4 contracts. They changed no production contract. Cancellation-induced loopback disconnects are expected and handled locally.

## Convergence

All seven functional requirements, five success criteria, three user stories and implementation tasks T001-T010 were assessed against code and tests. No missing, partial, contradictory or unrequested behavior remains in the implementation scope. No persistence migration, production inference, owner media, release or merge is included. Linux and alternate-backend/package validation remain pending exact-head hosted CI. The six contract checklist markers remain reviewer-owned; the preceding evidence answers their protocol, citation, bound and redaction questions without changing ownership.

## Publication boundary

Owner explicitly authorizes push, PR creation and at most two code/security review rounds, followed by final owner review and squash merge. PR state, exact-head CI and final thread audit will be recorded in its description after completion.
