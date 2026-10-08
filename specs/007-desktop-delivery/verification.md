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

Local native package acceptance and convergence are complete. Automatic push/PR and hosted platform/review results are pending. No S007 review request has been made yet.
