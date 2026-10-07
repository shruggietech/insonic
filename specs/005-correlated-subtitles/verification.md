# S005 verification record

## Scope and decisions

2026-10-07: S005, specs/005-correlated-subtitles and codex/005-correlated-subtitles deliver #21/#7. The original #7 roadmap title remains unchanged. Owner authorization covers push, official PR and review replies, with initial automatic review plus at most one combined manual second request. Stop before merge/release.

Cueson 1.2.0 governs the sole current embedded subtitle and assignment document. Known-speaker mappings are external and recording-local; consumers store current evidence references. Accepted replacement invalidates obsolete corpus preparation and queues physical retirement, preserving source media, supplied subtitles and elected model weights. Historical schema3 is frozen before schema4 migration. Large results are paged and digest-bound.

The packaged upstream schema is byte-identical at SHA256 f2661a3d52effbab4a82a4d47197b5c7fae58496dc30a397ea3f2f668358b654. Insonic schema checks retain strict validation; a separate upstream compiler accommodates its conditional type declarations without removing constraints or formats. Processing configuration has schema and pure runtime validation before opening tools/models. Concurrent mapping corrections require the expected mapping revision.

## Implementation and deterministic verification

2026-10-07: Go test ./... and go vet ./... pass. Full catalog/artifact race suites pass; full app race passes in 94.056 seconds, followed by passing focused races for the final reuse, cleanup and proof regressions. New regressions cover overlap multiplicity, unchanged-only untimed reuse, source/current/mapping fences, cancellation, stale snapshot rejection, current-reference invalidation, physical retirement, and retry after failed legacy readback. TestCloseCancelsInFlightCatalogCall now injects its blocking adapter before concurrent startup.

npm run check and npm test pass (54 Node tests). Registered contracts validate offline, including exact upstream Cue JSON. Python discovery passes nine tests; native qualification includes two media-tool, six fixture and ten processing-worker guard tests. No deterministic check runs inference, loads inference models or downloads weights.

Actual Windows native qualification passes pinned Cueson, SQLite/LadybugDB, real metadata/media extraction, all committed SRT/VTT fixture paths, PCM decoding, strict export refusal and source preservation. Production desktop bridge, hidden webview, offline help and cross-process CLI acceptance pass. Site build renders 27 pages and validates relative links/anchors. UTF-8 without BOM, LF, mojibake and version consistency checks pass. Git whitespace validation passes with native SRT/VTT terminal cue-separator blank lines allowed; its default blank-at-EOF check reports those four fixture files only.

PostgreSQL/S3 integration coverage includes current recording replacement, corpus invalidation, populated historical migration and legacy physical retirement. Integration tags compile locally. Hosted adapter fixtures pass on c12920a1f381be30b8d198b0b3818863695ae28f (CI run 37692666382, job 113036603097), exercising PostgreSQL/S3 alongside ArcadeDB. Linux/macOS native operations remain pending hosted evidence.

## Committed real-media fixtures

2026-10-07: Ordinary Git includes tests/fixtures/media/speech.flac (495373 bytes, 10.8 seconds, 48 kHz mono) and sintel-dialogue.mkv (4200055 bytes, 30 seconds, 426x182 at 24 fps, H264 plus lossless FLAC). Combined media size is 4695428 bytes. Both retain CC BY 3.0 attribution, source/rendition identity, exact hashes, crop/stream mapping, adaptation commands and independently published reference text. No LFS or test-time media fetch is required.

Speech SHA256: 62a3fcd36560793a065d53b5ea749dcf710d85028c837802063247ef787e38c3.
Video SHA256: f61adfa5641913a5b70e23f9dda2cef1012d9edc59ad895f6da7fd5c07874d6e.
Sintel character labels are editorial visual annotations, not measured acoustic ground truth. Direct auditory reference verification was unavailable.

## Explicit maintainer inference, outside CI

2026-10-07: scripts/qualify-processing.py explicitly runs elected local engines outside CI. CI environment guards reject inference before model materialization. Required checks use supplied subtitles/deterministic adapter results, real probing/decoding and actual Cueson only.

Measured Windows CPU engines: faster-whisper 1.2.1 / CTranslate2 4.8.2, tiny revision d90ca5fe260221311c53c58e660288d3deb8d356; pyannote.audio 4.0.7 / Community-1 revision 3533c8cf8e369892e6b79ff1bf80f7b0286a54ee. Existing local models were used; no new weights were downloaded. Worker SHA256: 5d071e50d79fb439fe352cee73e04a7d1a4c349a408a9e3fc2ca3fec11fd16a0.

Speech recognition: four segments, 4/33 word errors (12.12%), 0.724 seconds, approximately 230 MB peak RSS. Diarization: two voices/four turns, 5.25 seconds, approximately 1.23 GB peak RSS.
Sintel recognition: five segments, 2/31 word errors (6.45%), 0.729 seconds, approximately 231 MB peak RSS. Diarization: one voice/six turns, 22.14 seconds, approximately 2.72 GB peak RSS, despite the two-character editorial reference. No DER is claimed.

The actual product CLI completed six processing combinations across both fixtures (generate/run, generate/reuse, reuse/run), digest-bound reads and Cue JSON/SRT/VTT exports in 91.062 seconds within a 600-second deadline. Qualification checked scratch removal. Receipt paths under ignored build/native include processing-qualification.json and processing-product-0d8f8bc7/receipt.json. Measured CLI SHA256 was 6f3113382dd029ac7ab5ddfc372ea0729e2a27c531998112b943fd56e6420729. Later pure validation/CAS/reuse fixes have deterministic regression coverage; this receipt is not an exact final-commit engine run.

Inputs, output, threads, audio duration and wall-clock time are bounded; RSS is measured rather than hard-capped. Community-1 frame overrun within 250 ms is explicitly intersected with physical media bounds and diagnosed; generic malformed outputs are rejected. Linux/macOS/CUDA inference and measured diarization accuracy remain unverified. No hosted fallback occurs.

## Convergence and publication

2026-10-07: Pre-implementation analysis covered 14 FRs/five SCs with 36 tasks. Implementation convergence appended T037 (overlap/untimed reuse) and T038 (configuration parity); both are implemented and tested. Final code assessment covers 14 FRs, five SCs, 19 acceptance scenarios, the plan's current-record, mapping/reference, migration/proof, scratch, replacement, reuse, input-preservation and qualification decisions, and all five constitution principles plus repository/workflow constraints. No remaining code finding; existing publication/review/hosted qualification tasks were completed through the workflow below rather than duplicated.

Official PR: https://github.com/shruggietech/insonic/pull/24, published at implementation commit c12920a1f381be30b8d198b0b3818863695ae28f. Initial automatic Code Review completed with two P2 findings: optional UTF-8 BOM WebVTT format detection (4212432323), and exact required model-role casing on case-sensitive filesystems (4212432328). Both are fixed with regressions: default processing/assembly retain BOM WebVTT bytes through actual Cueson; canonical model roles are enforced before publication reads with case-alias collisions still rejected. Targeted race tests and Go vet pass. Fix commit f6fb5a73a2b53cbbd08524bb40e6efafc8d0c12b is pushed. Both findings have individual replies and resolved threads. Exactly one combined second code/security request was posted: issuecomment-6047748617. Second Code Review completed on f6fb5a7 with one P1 finding (4212506107): new local workers/Cueson inherited unrelated plaintext credentials. The finding is fixed with explicit clean environments in Capture/Run, new local FFmpeg/Cueson/worker calls and maintainer children. The allowlist preserves system/locale and selected native-library/device variables, not ambient credentials or Python/home configuration. Workers receive only a new secured empty HOME/USERPROFILE, deleted after execution; pyannote requires this home even in offline mode. All six hosted checks passed on f6fb5a7. No separate Security Review report/run is visible despite the combined request; no additional request is made because the review cap is exhausted. Final fix ebae853a1a64b559f0b5a07474c8a39d6d2a3d6b is pushed, has an individual reply (4212627205), and its discussion is resolved. No bot approval of the final fix commit is claimed. The owner performs final review and the authorized squash merge ritual.

2026-10-07: Post-review environment regressions fail against the original inherited environment and pass with isolation. Actual children verify omitted session/provider/Hugging Face credentials, retained explicit flags/loader selectors and private-home cleanup after success, failure and timeout. Thirteen Python worker/qualification tests and process/processing race suites pass. Exact native Cueson and BOM processing/assembly race scenarios pass with the clean environment. Full Go tests/vet, 54 Node tests, schema/text checks and site build pass; the final private-home delta passes affected tests/vet. No inference runs in these required checks.

2026-10-07: Explicit Windows CPU clean-environment qualification reruns both committed fixtures with existing models and unchanged worker identity. Speech recognition: 0.725 seconds/about 229 MB RSS; diarization: 5.51 seconds/about 1.22 GB. Sintel recognition: 0.701 seconds/about 230 MB; diarization: 19.44 seconds/about 2.71 GB. Recognition errors and diarization split/merge limitations remain as reported above. The ignored processing-qualification.json receipt is refreshed. This is maintainer evidence outside CI, with no new weight download.

## Owner handoff

2026-10-07: All six affected CI gates pass on final product-code commit ebae853a1a64b559f0b5a07474c8a39d6d2a3d6b, run https://github.com/shruggietech/insonic/actions/runs/37695813342. Foundation 14 seconds; documentation 30 seconds; adapter fixtures 1m5s; Linux native 2m41s; macOS native 2m50s; Windows native 4m41s. Every job remains below ten minutes and no required check runs inference or downloads weights.

All three external findings have pushed corrections, meaningful regressions, individual replies and resolved discussions. Two Code Review rounds completed (initial automatic plus one combined manual code/security request); no third request occurred. A separate Security Review report/run did not appear, and final-fix bot approval is not claimed. The final receipt commit changes only task/verification records; the final PR head is confirmed green in the PR body before the owner handoff.

S005 completes all 38 tasks and delivers all four stories for #21/#7. PR24 is ready for the owner final review and squash merge after final receipt CI. No merge, branch deletion or release is performed by this slice. Measured engine-quality and platform/device limits remain as recorded above.
