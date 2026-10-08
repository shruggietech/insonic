# S007 verification and publication

Date: 2026-10-07. Branch: `codex/007-desktop-delivery`. Issue: #10. Version: 0.0.0. Base: `bd00e15e5971de6135c7f5d5c1c66f94f46a441d`.

## Scope and authority

Spec Kit specify, clarify, checklist, plan, tasks, analyze, implement and converge apply to this slice. The owner explicitly authorized push and official PR publication, at most two review rounds, and handoff before owner merge. No release/tag/promotion is authorized. Required verification uses no recognition/diarization inference, model initialization/loading, weight downloads or required engine output.

## Local implementation checks

- Repository integrity/schema checks pass, including UTF-8 without BOM, LF, mojibake, links, version 0.0.0 and governed kit hashes; 19 schemas, 17 contracts and 21 examples.
- Root Node suite: 55 passing tests.
- Full Go acceptance and vet pass after shared playback lifecycle integration. An earlier run overlapped an incomplete source edit and failed compilation; the complete implementation was rerun successfully.
- Affected Go race checks pass for app, artifact, schemas, CLI and native desktop; the full installed-companion package race suite also passes.
- Frontend typechecking, static build and 27 rendered interaction/schema tests pass. Outgoing rendered requests are checked against the published contracts; native nested speaker identities, mixed timed/untimed evidence, mapping pagination CAS and assembly recovery have focused cases.
- Python acceptance: 14 passing tests, including five package archive/inventory/discovery cases; separate media pin suite: four passing tests, including actual pinned-source archive extraction.
- Public and offline documentation exports each validate 27 pages and local links/anchors.
- Exact current clocks, byte-preserving raw captures, settings conflicts/repair, immutable/current playback binding, bounded snapshots, lease preparation renewal, expiration/release/crash cleanup and wrong-workspace rejection have focused regressions.
- Real pinned FFmpeg/FFprobe regressions import AAC/M4A and FLAC/Matroska renditions with nonzero source clocks. Normalized previews are compared against complete decoded PCM and a 0.25-second cue window; preview duration and zero start are verified. Original-source seek offsets derive from measured source/proxy clocks.
- Real pinned Cueson assembly of committed speech/subtitles passes for supplied turns at 0–1 seconds and 2.5–2.75 seconds. Optional absent processing tools serialize as omitted fields, while explicit partial pins still fail validation.
- The actual Windows WebView2 source journey passes all twelve mounted-control flags: audio/video import and playback advancement, metadata/date/relocation, current assembly, cue/span seek, terms, persisted speaker/aliases, pipeline inspection, durable jobs, appearance and keyboard/help. The receipt records `model_execution: not-run`. Native bridge playback requests now omit absent optional document bindings and are schema-validated in regression fixtures.
- The final relocated Windows package passes sibling CLI, actual media import, Cueson assembly/export, installed companion discovery, help and native webview. Its webview receipt records all journeys and audio/video playback passed, with model execution not run. Linux/macOS execution and hosted timing remain PR CI gates.
- Failed/interrupted supplied-turn assembly refuses generic retry before durable mutation and instructs input resubmission. Actual catalog regressions preserve the entire original work record; ordinary import retry remains covered.

## Convergence

The first convergence assessment checked 15 functional requirements, five success criteria, 12 story acceptance scenarios, seven plan decisions and five constitution principles. It found two HIGH gaps: buffered-media revocation after another client's replacement (partial, FR-009/FR-010) and repeated native package CLI polling processes (contradicts the project execution constraints). T029 now revalidates mounted tickets every five seconds and stops obsolete buffered playback with rendered/native regressions. T030 now waits through one bounded CLI process using the real shared work record and IPC, with terminal-state and timeout regressions.

Final convergence checked the same inventory and found zero missing, partial, contradicting or unrequested implementation gaps. The convergence assessment left tasks.md unchanged (SHA-256 `874f019128aff6f1e71615303522f61ca43005aeb5a541773bd00cfce54bf0e8`); implementation completion marks were recorded separately afterward. Hosted three-platform package execution, sub-ten-minute timing and exact-head checks remain publication acceptance gates, not claims inferred from Windows.

## Native delivery boundary

Windows x86-64, Linux x86-64 and macOS ARM64 layouts include CLI/GUI, Cueson/schema, metadata/media tools, native libraries, offline help, notices and exact inventories. Installed discovery remains relative and never persists installation paths into portable settings. Explicit workspace configuration retains precedence.

Archives remain unsigned and unpublished. CI publishes receipts/inventories/checksums rather than binary archives. Windows/Linux GPL-enabled companions require complete corresponding-source delivery, including enabled external libraries, before public release distribution. The exact-source macOS build disables GPL/nonfree/autodetected external libraries and explicitly enables system VideoToolbox. General graph/exploration, assistance, portability/release promotion and training are outside this slice. Real model/provider execution is not claimed by these checks.

## Publication and reviews

Local native package acceptance and convergence are complete. Windows package construction plus extracted acceptance took approximately 120 seconds (99.375 seconds for extracted acceptance, including 31-second detached-owner cleanup). The package contained 1,534 files; archive SHA-256 `529a4b90942edfca523aeb818c1e661ded7714519d563f1abe6588e4e220f855` and inventory SHA-256 `bd0a5186580ed36fb8dc0b973c3e986502869231e17da9f4d218d5206d2cc0cd` identify the local source-dirty qualification archive.

Official PR: https://github.com/shruggietech/insonic/pull/26. Initial implementation head: `a15a2af5fa8b5b8ec0fb67655c1ebee9d6862629`. Issue #10 is In review and the PR is attached to the Codex task. At initial publication, automatic code review and hosted checks were pending and no second-round request had been made. Later closure evidence follows chronologically below. Owner merge is the stop boundary.

### First review and hosted acceptance

Initial automatic review completed with two findings. P1 required installation-relative Unix native-library loading and a relocated check that cannot fall back to the checkout. Package linking now uses `$ORIGIN/native` or `@loader_path/native`, normalizes macOS library identities and validates loader metadata. Extracted acceptance isolates and restores the original native library directory. P2 required separating effective package tools from persisted overrides; effective values are now read-only, explicit editors do not copy package paths, and shared `settings.set` with a null value performs a CAS reset. Rendered UI and real installed-origin helper tests cover default save suppression, reset, relocation and conflict behavior.

Self-review also fixed optional FFmpeg serialization, with persisted-schema and reload tests that failed before the change. Full Go acceptance/vet and affected races pass; frontend typecheck/build and 33 tests pass; packaging tests are seven passing and source-pin tests four passing.

Initial hosted foundation/docs/alternative adapters/Windows native checks passed. Windows completed in 7 minutes 14 seconds. macOS compiled exact-source media tools in 157 seconds but a real fixture test required the missing built-in FLAC encoder; the source configuration now includes it. Linux timed out before any frontend receipt. Classic deferred IIFE loading, explicit deduplicated fixture readiness, bounded progress and a cryptographic UUID fallback address native custom-scheme startup compatibility; actual Linux execution remains pending.

The revised Windows package passes all extracted/native journeys with checkout libraries absent throughout acceptance and restored afterward. Acceptance took 100.656 seconds including cleanup, approximately 122 seconds with construction; 1,534 files, archive SHA-256 `e936907c76de9ef25d658f4324ec95b98502afd6bc522cd2d3eb6ab3feccf09b`, inventory SHA-256 `3cec6accd8164f06f30004508a83e750cadaebac107cd29f9662b36b72596466`. Unix native execution and the single authorized second code/security review remain required.

### Second review and native transport correction

Both first-round threads were replied to and resolved at `4c6b4ca`. The single authorized manual second round requested code and security review in comment `6050801421`. The code review completed at `4c6b4ca` with one P2 finding: Settings did not load the persisted credential backend. Shared `settings.show` now reports that backend without replacing the manager; Settings loads it, disables unchanged selection and rereads after deliberate changes. Actual vault/session reopen tests preserve persisted bytes and selected mode; default-native and populated-session checks pass. Rendered regressions verify no mutation on reopen or failed read. A separate security report has not appeared; no additional request will be made.

Hosted run `37716981523` passed foundation, docs, adapters and Windows native (7 minutes 53 seconds). macOS exact-source media compilation and real regressions passed in 154 seconds. Linux reached audio playback but could not load metadata; macOS reached video after successful audio and cue seeking, then timed out. Native custom-scheme media limitations prompted the documented loopback transport correction. The existing handler now streams through an ephemeral 127.0.0.1 listener with strict authority checks and process shutdown. No browser Blob, media-size reduction or external routing was added. Source and packaged macOS app metadata share an IP-specific ATS exception.

Post-correction local full Go acceptance/vet, desktop race checks, 55 root tests, 36 frontend tests/typecheck/build, 17 Python acceptance tests and repository integrity/schema checks pass. Real Windows native mounted-control qualification passes with the loopback transport. Hosted Linux/macOS source and extracted-package execution, final exact-head CI and second-round thread closure remain pending.

### Completed implementation and review acceptance

Implementation head `f2714ab13f2d6f1485be0bd10a4e7689fd735b11` passes every job in hosted run `37718362131`: foundation 15 seconds, docs 25 seconds, alternative adapters 1 minute 3 seconds, Windows native 7 minutes 40 seconds, Linux native 6 minutes 41 seconds and macOS native 6 minutes 12 seconds. All required jobs stay below ten minutes. Source and relocated native packages execute real audio/video playback and the complete mounted-control journeys. No transcription/diarization inference, model initialization/loading or weight downloads are required.

The fresh local package at that clean implementation head passes every extracted check with checkout libraries isolated and restored. Smoke acceptance takes 98.735 seconds; 1,534 files, 130,247,062-byte ZIP, archive SHA-256 `856155ddd39988e00c10f9d4fa4d7a55cb61f3e32612303b24e0f7ea7bab3f59`, inventory SHA-256 `f7e398a99ca78a104451fa1f885d4eb8998adddb483d2a7f538fab68dc353d71`.

Exactly two code-review rounds completed: automatic at `a15a2af`, manual at `4c6b4ca`. The second request also named security review; no separate security report or approval appeared. All three received findings have changes, passing regressions, linked replies and resolved threads: first-round P1 `4213854793` and P2 `4213854807`; second-round P2 `4213959580`, replied in `4214014307` after `f2714ab`. There are zero unresolved threads. No third review request was made.

Post-review convergence assessed 15 functional requirements, five success criteria, all 13 story acceptance scenarios, seven design decisions and five constitution principles. No remaining missing, partial, contradicting or unrequested implementation gap was found. The assessment left tasks.md byte-for-byte unchanged at SHA-256 `273b06bfe26452aec1e44c1598444f803124798d3a8c9762c50c4ee8d3f9dba6`; completion marks were recorded separately after acceptance. Final record-only publication receives exact-head CI before the owner handoff. Issue #10 remains In review until the owner's squash merge; no merge, release tag, binary promotion, signing or notarization is performed here.

### Receipt provenance audit

The three hosted receipt sets validate source and extracted media/Cueson journeys, loader isolation and inventory hashes. Their build revision is the synthetic PR merge `7dc154859bcabdc9aa8cb7ff6041d59e1fb1b082`, whose parents are base `bd00e15` and implementation `f2714ab`; macOS cold media compilation took 123.44 seconds. Detailed receipt inspection identified a further reproducibility gap: FFmpeg's source version script discovers the enclosing Insonic checkout Git repository and reports that merge abbreviation rather than pinned upstream 7.0.2. The source archive identity is correct and behavioral checks passed, but the reported version requires T037 correction and cache invalidation before final handoff.

T037 validates the extracted RELEASE file against pinned 7.0.2 and writes the upstream-supported VERSION override before compilation. The release/version policy now participates in both the cache key and receipt; cached and fresh companions must report 7.0.2. An actual hash-pinned upstream `ffbuild/version.sh` regression inside the consumer checkout reports `bc16ecf` before the override and `7.0.2` afterward. Five offline media-pin tests pass, including release mismatch refusal. The source commit/archive digest remain unchanged. No further gaps were found in the receipt audit; the updated source-build policy receives a fresh hosted macOS compile and final exact-head checks before handoff.

Hosted extracted acceptance at the preceding implementation head is independently recorded below. Every inventory file hash was validated, source_dirty is false, and native libraries were isolated and restored on every platform.

| Platform | Included files | Archive bytes | Extracted acceptance seconds |
| --- | ---: | ---: | ---: |
| Windows x86-64 | 1,548 | 130,215,016 | 74.140 |
| Linux x86-64 | 1,336 | 146,121,920 | 51.722 |
| macOS ARM64 | 1,337 | 94,791,561 | 44.994 |

The final handoff records the exact final PR head/check run in the PR after all required checks finish. Review status remains closed with two requested rounds; no third request, owner merge or release promotion is performed.
