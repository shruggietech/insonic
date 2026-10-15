# Verification: S032

## Local verification, 2026-10-15

Spec Kit specify, clarify, checklist, plan/research, tasks, blocking analysis and implementation executed. No extension hooks installed. Review-owned requirement-quality markers were evaluated in analysis without changing their meaning. Convergence identified one browser-cleanup gap, appended T014 and implementation added an owned-server kill fallback and stalled-close regression.

- Root npm run check: passed text/BOM/LF/mojibake, Markdown, links, JSON/YAML, version/schema and governed-kit integrity.
- Root npm test: 111 passed before the final cleanup test; final rerun: 112 passed.
- Site npm test: 14 passed.
- Site build: passed public/offline export checks (111 expected pages each).
- Python suite: 292 passed before the final publication tests; final rerun: 299 passed in 71.122 seconds.
- Publication tests include invalid current/newest CI, missing actual required jobs, archive/receipt failure before transport, immutable schema bytes, release metadata drift, non-force conflicts/uncertain responses, and real local binary Git tree/replay.
- No local installer or browser was launched. All local console children use verified hidden launch patterns. Test browser closure uses a fake retained server handle.

## Pending hosted evidence

Full final-head native/package/renderer CI, complete timing below 660 seconds, optional read-only production content/download/eight-browser proof, and at most two external review rounds remain delivery gates. Production deployment using the new workflow and merge remain owner boundaries.

## Hosted first pass

PR #108 opened at 311528c. Initial read-only smoke verified complete deployed bytes and all five package responses, then rejected hydrated /docs/latest/ because the initial browser assertion accepted only the exported index.html spelling. Both paths select the same page. The checker now fetches and digest-checks the actual directory alias before accepting it; no layout or content check was removed. A deterministic single theme initializer replaces repeated init-script registration. Duplicate native work from the superseded smoke was cancelled; explicit maintenance smoke now leaves full qualification to the independent PR run while retaining foundation/docs.

## Hosted Windows qualification recovery

Run 38764795007 passed every elected lane except relocated Windows Desktop ZIP qualification. The owned timeout receipt records poll_before_kill null and cleanup_timeout false: the executable was still alive, not merely a pipe inherited by another process. It had emitted the WebView success JSON before deferred cleanup. Targeted fixture-only shutdown checkpoints and a pre-watchdog goroutine dump distinguish runtime cancellation, fixture removal, playback closure and native process exit. Success receipt emission now follows all cleanup. The 90-second process deadline and all native/MSI checks remain intact. No third review round is requested.

## Measured cause and repair

The instrumented run 38767257627 reproduced the timeout earlier in the journey. Its 85-second stack shows packageenv.verifyFile inside ReadMediaTools, reached from libraryService during an ordinary media catalog operation; Wails was still in its message loop and no cleanup checkpoint had run. This corrects the initial shutdown hypothesis. Both installed CLI and GUI repeatedly hashed the complete companion closure for catalog-only views and background cleanup, while source builds bypassed installed discovery.

A failing application regression confirmed media.list was blocked by an unrelated malformed executable configuration. libraryCatalogService now serves catalog-only reads, origin/location updates and artifact cleanup without resolving media executables. Import, refresh, processing and actual tool validation retain their fresh hash checks; no integrity cache or metadata-based trust shortcut was introduced. Processing executor construction uses the already-elected tools and artifact service without a redundant media-tool read. Temporary goroutine-dump instrumentation is removed after diagnosis; truthful post-cleanup success receipts and bounded fixture-only checkpoints remain. Full hosted rerun is required.

## Frozen screenshot gate recovery

Run 38769605433 correctly stopped downstream native jobs when the documentation producer rejected changed backend files against an older application screenshot receipt. All 1,252 recorded application source hashes independently match captured main ancestor cf28029a43f3c3e6a7f3d9ab235677874087633e. The released product tag differs in two qualification-only files and is therefore not substituted for the recorded capture revision.

The builder now retains the current-byte fast path and otherwise validates every source blob plus VERSION against that exact ancestral revision using a bounded hidden Git batch. Missing/foreign revisions, changed hashes or wrong versions fail. PNGs, captions, dimensions and receipt bytes remain unchanged. Full history is fetched by producers; native CI still captures current application UI. A historical-capture regression failed before the change and passes afterward; the complete local public/offline build passes with frozen inputs intact.

## Complete native repair and remaining startup failure

Run 38771007634 on 90e17ae passed relocated Windows ZIP, source-built Windows UI, CLI, both Linux package variants, core suites, adapters and documentation. MSI lifecycle assertions passed, but the first authored-page worker exceeded its 15-second restricted-session initialization cutoff (renderer phase opening-restricted-session). Candidate aggregation correctly withheld publication eligibility. This is a distinct native capture startup failure, not the repaired catalog/tool-discovery timeout.

Windows package setup waited for the source producer and then spent 71 seconds in Go setup, 30 seconds installing compiler dependencies and 36 seconds installing notice inputs. The package consumer now overlaps that independent setup with source preparation. A bounded read-only artifact wait selects only the current run/attempt; the existing download digest and full source restore checks remain. Candidate aggregation explicitly requires producer success. Release callers retain their original dependency scheduling. Capture startup gets one fresh-worker retry only for the exact observed initialization timeout; assertion failures and repeated initialization failures still reject the receipt. No per-worker or job watchdog increased.

Local follow-up checks passed: root npm tests 117, installer model tests 58, text/schema checks (1,269 files) and complete public/offline export build. These tests run through the verified hidden launcher; native APIs in installer unit tests are mocked. The latest hosted run already demonstrated the catalog repair and frozen-capture build. T017 and complete workflow timing remain pending.
