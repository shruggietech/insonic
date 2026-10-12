# S020 verification

## Scope and behavior

Issues #70 and #62 share saved import preset lifecycle, deterministic admission and release-matched executable documentation. The current 1.0.1 candidate remains unreleased; historical 1.0.0 documentation/contracts are unchanged.

Presets persist nonsecret destination/date/pipeline choices with optimistic revision checks, immutable replay receipts and tombstones. CLI and desktop share catalog operations. Import resolution applies preset < manifest defaults < item < explicit request overrides, preserving false, zero and explicit pipeline opt-out. Frozen selections retain preset revision, immutable destination and elected processing configuration. Accepted items reconcile one deterministic processing child under a parent lease/generation fence. Missing/deleted/ambiguous/incompatible selections and unready pipelines fail before enqueue. Exact-title recording targets retain supplied-text behavior. Identical batch elections resolve once without rewriting existing profiles.

## Local automated evidence

All commands used hidden, noninteractive process creation.

| Check | Result |
|---|---|
| Full Go suite, go test ./... -count=1 | Pass on Windows, including app/catalog/artifact/library/backup |
| go vet ./... | Pass |
| Integration-tag catalog/artifact/library/app/backup compilation | Pass; live PostgreSQL/S3 fixtures delegated to required CI lane |
| npm run check | Pass: UTF-8/LF, links, JSON/YAML, version/schema and upstream brand integrity |
| npm test | Pass, 84 tests |
| Python unittest discovery | Pass, 141 tests |
| Frontend TypeScript and mounted regression suite | Pass, 73 tests |
| Site build and current executable JSON fidelity | Pass, 78 pages in each online/offline export |
| Extracted Windows CLI, all qualification journeys | Pass, 148 contextual help paths, documented workflows and licensed/synthetic media |
| Extracted Windows desktop, all qualification journeys | Pass, CLI workflows, native bridge/webview and mounted preset create/edit/select/import/delete |

The local package receipts at build/native/package-cli-receipt.json and build/native/package-desktop-receipt.json identify the base revision and source_dirty=true. They are development-tree evidence, not exact committed-head release artifacts. Final small convergence changes were additionally covered by the full Go/contract/frontend/site checks; required CI requalifies the final committed source on all three platforms and both package variants. The successful desktop smoke took 269 seconds.

New regressions cover workspace isolation, reserved-setting bypass, secret rejection, stale revisions, replay after edit/delete, tampered profile restore, alternate filesystem publication/read/lease/retirement, S3 routing, preset resolution and explicit precedence, frozen processing/destination retry, cancelled-parent fencing, incompatible versions, name/ID ambiguity, exact-title supplied text and deduplicated batch resolution.

Documentation qualification reads marked JSON directly from current authoritative pages. It executes preset CRUD, direct/JSON/CSV imports, refresh, pipeline inspection/processing, speaker creation/mapping/correction, extraction/query/save/layout/calendar, dataset/training/fetch/profile and source configuration. Deterministic HTTP fixtures prove routing, multipart and result effects; they do not measure acoustic model quality or a live third-party service.

## Convergence

Analyze mapped all 12 functional requirements to implementation tasks with no unresolved findings. Converge assessed 12 functional requirements, 5 success criteria, 10 acceptance scenarios, 5 design decisions and 5 constitution principles. Three partial gaps (two high, one medium) produced T021..T023; their regressions failed before repair and pass afterward. Follow-up source assessment finds no remaining buildable gap. No unrelated product-use gates, release publication, guided setup or managed-worker feature was added.

## Remote delivery

Official PR, external review rounds and final-head CI are pending publication. Maximum two review rounds are authorized. The owner retains final review and squash merge authority.
