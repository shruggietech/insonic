# S010 verification ledger

## Spec Kit gates (2026-10-09)

Specify, clarify, requirements/domain checklist, research, plan and tasks executed with installed skills/helpers. No extension hooks are installed. Branch is codex/010-canonical-annotated-import; feature directory is specs/010-canonical-annotated-import; completion targets are #31/#32, with #30/#33 foundation scope only.

Analyze read all specification/design/task requirements after the owner approved D1-D4 and the separately authorized constitution amendment to 2.2.0. Coverage: FR-001..017 and SC-001..005 map to T004..T024; all three user stories have failing-test, implementation and integration tasks. No unresolved critical/high findings. Concrete risks (composite receipts, arbitrary-ID downstream bounds, locator scrubbing, canonical stream remap and native MP3 capability) have explicit tasks and verification. Deferred whole-product work is documented rather than claimed done.

Checklist gate: requirements 6/6; admission 6/6 requirements-quality review passed. Markers do not claim implementation completion. Existing ignore files cover build/runtime/private credentials; no additional publish ignore file is required for this private maintainer package.

## Baseline

Affected Go tests passed before implementation: internal/subtitles, internal/library, internal/catalog, internal/app. This was the pre-change baseline.

## Implementation and publication

Implementation evidence follows. User explicitly authorizes automatic push and official PR, initial external reviews and at most one second @Codex round. No merge, tag or release is authorized.
## Convergence pass 1

Checked 17 functional requirements, 5 success criteria, 11 user-story acceptance scenarios, 12 plan decisions and 5 constitution principles. One HIGH partial finding: standalone target resolution occurred at worker start, so delayed replacements could observe a newer document than submission. Appended T027 under Phase 7; implementation continues. No missing, contradictory or unrequested build findings in the other assessed boundaries. Platform CI and publication remain verification/delivery obligations.

## Implementation evidence (2026-10-09)

| Boundary | Evidence |
| --- | --- |
| Atomic current admission and portable proof | Composite catalog tests cover one revision, rollback, stale/cancelled authority, accepted replay and forged snapshot rejection; PostgreSQL uses the same suite in adapter CI. |
| Document preservation and local identity | Packaged exact 1.0.0/1.1.0 schemas plus six original-envelope fixtures; 1.2.0 direct preservation, integer-safe translation, Unicode/case/origin grouping, all non-clobber modes and bounded local token tests. Mapping and fresh restore retain non-UUID IDs. |
| Transcript targeting and retirement | Standalone selector/collision/replacement, invalid replacement, frozen queued revision, independent HTTP budgets, managed-sidecar conversion, shared current references and lease-aware delayed retirement. |
| Canonical media | Real pinned FFmpeg qualifies stereo FLAC, grouped languages/offsets, passthrough MP3, AAC-to-V0, exact nonzero 32-bit integer PCM and six-channel preservation. Native embedded/explicit precedence and Cueson SRT/VTT/ASS/SSA ingestion pass. |
| Recovery and locator retirement | Accepted-response loss reconciles without inputs; rejected randomized container candidates retire before recomputation; accepted items scrub locators without breaking replay identity; native runtime partial-batch retry passes. |
| Processing/playback | Selected grouped-track native playback preserves original offsets and complete duration; current supplied text bypasses recognition; explicit diarization uses deterministic adapters in required tests. |
| Contracts and interfaces | CLI flag/manifest precedence, 53 rendered desktop tests, TypeScript/build, schema examples and repository checks pass. Version remains 0.0.0 across software/docs/master schema. |

Checks completed locally: npm run check (628 text files, 20 schemas, 18 contracts, 22 examples), npm test (57 tests), full Go test and vet, affected catalog/library/subtitles/app race suite, final native library/app tests, frontend check/test/build, documentation export (27 pages with resolving links/anchors), native dependency qualification, cross-process CLI qualification, desktop bridge and hidden native webview journeys, and five source-packaging tests. Final replay/frozen-target changes additionally pass focused race tests. Required verification performs no recognition, diarization, matching or training model execution.

No claim of Linux/macOS or server-backend local execution: platform-native/package and PostgreSQL/S3/ArcadeDB fixture verification is assigned to PR CI. macOS source-built static LAME and its corresponding-source/rebuild packaging require that CI evidence. Floating-point PCM and more than eight channels per FLAC stream fail visibly before current acceptance because the approved FLAC encoding cannot preserve them through this pinned path. Full audio replacement and bulk legacy-audio migration remain #30/#33; no installed release is promoted here.

## Convergence passes 2 and 3

Pass 2 found missing actual-audio interval validation (HIGH, FR-013 and #31 alignment acceptance), appended T028, and resumed implement. The preservation path now checks timed consumer claims using rational source-clock audio bounds, distinguishes unknown from zero, and leaves native cue conflicts and untimed participation intact. Focused zero/nonzero/unknown fixtures pass.

Pass 3 rechecked all 17 functional requirements, 5 success criteria, 11 story scenarios, 12 decisions and 5 principles. No remaining build findings. T027/T028 are implemented and verified. T021/T024 remain pending platform/backend CI evidence; T026 remains pending official publication/reviews/exact-head CI. No further convergence task is needed for delivery operations already tracked by those tasks.

## Publication and review record

Official PR #36 published at b953919. Initial automatic review is round 1; at most one explicit @Codex second round is authorized. Every finding needs an appropriate reply and direct thread-resolution verification. Final handoff requires green exact-head CI. Merge/tag/release remain owner-controlled.

Round 1 completed on b953919 with one P2 manifest-schema finding (PRRT_kwDOU8_K3s6qt-PI): overlapping input shapes were schema-valid but runtime-invalid. Replaced item alternatives with exclusive media versus standalone shapes, preserved record+source compatibility, prohibited duplicate transcript aliases, and required a target for observed revisions. Added positive/negative schema regression cases. Initial foundation/docs/core-Windows/adapter CI passed; native platform checks were still running when the repair was prepared.
