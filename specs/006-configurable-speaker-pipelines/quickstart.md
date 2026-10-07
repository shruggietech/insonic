# S006 validation guide

1. Use a clean temporary workspace with synthetic credentials and deterministic local/HTTP stage fixtures. Configure media/Cueson tools from existing qualification paths.
2. Create/inspect Local, Connected and Custom definitions, update with expected revision and reject stale/sensitive/invalid routes.
3. Import the committed audio/video fixtures and process supplied or deterministic generated results. Verify independent reruns, elected configuration/context and replacement/retirement.
4. Create a known speaker with aliases, map two recordings, select current speaker evidence and correct mapping. Confirm embedded document digests stay unchanged and dependent preparation invalidates.
5. Create active/scoped/inactive terminology, compile context twice and verify identical hints/digest, omissions and supported forwarding.
6. Exercise overlap/untimed/no-speech diagnostics, configuration defaults and mandatory document checks with optional quality disabled.
7. Run Go tests/vet/race as affected, deterministic Python tests, npm check/test, site/offline build and real CLI/desktop qualification. Hosted CI proves three OSes and alternate backends without engine invocation.
8. Commit/push official PR, address every review and resolve threads, request at most one additional round, then verify final-head CI green and hand off before merge.

Actual recognition/diarization engines and weight retrieval are never required checks. Optional maintainer inference requires explicit election outside CI and is reported separately.
