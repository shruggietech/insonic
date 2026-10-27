# S052 verification

## 2026-10-27 planning and test-first work

Spec Kit specify/clarify/plan/tasks/analyze completed. Built-in quality checklist 16/16; 8 FRs, 3 SCs and 7 acceptance scenarios map to 11 tasks. Five constitution principles and eight plan decisions have no blocking conflict. Two read-only research agents resolved platform and fixture integration questions. No custom reviewer-owned checklist or extension hooks. Initial actual-terminal regressions fail at the missing transport import, as expected before implementation.

## 2026-10-27 local acceptance and lifecycle refinement

Actual Windows ConPTY probes pass all terminal handles, terminal-byte Ctrl+C, overflow refusal, forced-cleanup refusal, UTF-8/control validation and inherited deadline expiry. Per-launch invalid standard handles avoid inherited parent pipes without changing global handles. The maintained native-source CLI campaign passes all eight terminal cases with executable SHA256 690a88a8ff5e0c27e3b22212f2edf7c65c116d4622926a575d0ea296ace1ff71. JSON/quiet/redirected clients each record a correlated running diagnostic before the held acquisition releases. Ctrl+C naturally exits1, independent durable work remains running and later succeeds; timeout exits4, explicit cancellation and checksum failure exit1. All terminal clients, the idle runtime, listener/handlers and generated fixture directory release. Case durable_state records the independent final work outcome, not an interrupted client snapshot.

Full Python629 tests pass in45.182s with3 declared skips. Root205/205 and site17/17 pass; root integrity validates1654 text files,25 schemas,22 contracts and30 examples. Public/offline export validates111 pages. The first restricted test attempt encountered temporary-root permissions; unchanged tests use the approved hidden launcher, owned temp and verified toolchain/locale. Full-suite dependent package fixtures were updated for the required terminal contract. The generic CLI attempt lacked native graph capability; supported native qualification passed. The maintained deadline inventory initially identified four unclassified owners; all now have explicit expanded/shared or injected-failure dispositions. No product runtime or ordinary deadline changed.

## 2026-10-27 implementation convergence

Converged after checking all18 FR/SC/acceptance items, eight plan decisions and five constitution principles. No remaining implementation gap or unrequested product behavior. Exceptional descendant-observation/console-close cleanup now still releases handles and preserves the primary failure; seven transport regressions pass. Final delivery tasks remain open until final-source CI, review completion and retained-evidence cleanup. No empty convergence phase was appended.

Final Python suite after exceptional-cleanup regressions:631 tests pass in44.236s. Declared skips are actual Linux nested-session cleanup, POSIX descriptor inheritance and Windows short leaf names disabled on this volume. Repository/site results remain as above; final-source hosted platform checks are still required.

## 2026-10-27 post-publication integration convergence

Inspection found that newly explicit terminal/runtime roots bypassed the existing archive-lane CPU policy. Added T012 without rewriting the earlier convergence result. The elected factory now carries terminal creation through the same suspended root assignment/verified mask path and accounts for every root, while unsupported elected factories fail before allocating fixtures. Actual Windows terminal mask and unchanged parent affinity pass, alongside all seven transport cases; shared factory propagation and refusal regressions cover integration. No ordinary product operation changed. Initial CI remains at2277272 until its compact attempt audit is retained before the correction push.

Initial hosted Linux source and package failures all report runtime_endpoint at the first workspace show. The fixture placed XDG_CACHE_HOME below its deep library path; runtime endpoints append insonic/runtime plus a32-character hash and .sock, exceeding the existing100-byte IPC bound. T013 gives only the owned private host profile a short independently owned temporary root, while the library keeps its deep Unicode path. Teardown verifies both roots are removed. No product socket limit, operator path or timeout is relaxed. Initial attempt and failure evidence remain retained before the correction push.

Correction verification:633 Python tests pass in48.185s with the same three explicit host skips; root205/205 and integrity/schema checks pass. The separate short host cache is removed on failed-launch regressions. Elected Windows terminal affinity is verified before resume without changing the parent, and generic factories cannot silently bypass terminal policy. Hosted Linux execution remains a required final-source gate.
