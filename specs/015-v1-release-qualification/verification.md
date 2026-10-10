# S015 verification and delivery record

## Implemented behavior

Prepared an unpublished 1.0.0 candidate with synchronized owned software/runtime, package metadata, documentation and 23 current JSON schemas. Preserved all 46 baseline documentation/schema files byte-for-byte and kept upstream Cueson identities independent. Preparation no longer falsely marks a version released or rewrites generic baseline checker fixtures. Frozen reference validation follows the original repository link base used by the renderer.

Candidate validation compares packaged source pins, recipes and three build helpers against the selected checkout and requires every dependency group's verified receipt. CI retains both native variants on all three platforms, packages matching documentation and collects the exact same-run/attempt candidate with read-only credentials. Runtime and package lanes are independent; existing native, credential, bridge and alternative-backend checks remain.

The connected qualification harness drives actual shared CLI operations in disposable owned workspaces with bounded hidden subprocesses. Both modes require explicit maintainer election outside CI. It verifies native graph availability, exact nanosecond dates, current document bytes, model output hashes, rosters and query results after self-contained restore with the original workspace unavailable. Failures retain bounded structured codes, excluding raw exception content.

Qualification exposed and corrected nanosecond import schema rejection, valid embedding `input_count` provenance rejection and a pinned Whisper final timestamp overrun. The latter now has an explicit, diagnosed maximum 250 ms end/media intersection. Generic adapter timing remains strict. Regression failures were observed before their corrections.

## Local connected and acoustic evidence

- [Controlled journey](evidence/controlled-journey.json): PASS, 80.844 seconds, actual media/Cueson/LadybugDB plus deterministic acoustic adapters. No model accuracy claim.
- [Real Windows CPU journey](evidence/real-journey.json): PASS, 130.375 seconds, current pinned faster-whisper/Community-1 processing, embedding-profile enrollment, matching and exact recovery/query comparisons. CLI/tool/model/worker and working-tree input hashes are recorded. Local evidence identifies its tested working tree, not a clean CI revision.
- The cropped speech recognition initially failed with `invalid_timing`: its observed [5.36, 8.08] endpoint exceeded eight seconds of decoded audio by 80 ms. The final run retains `whisper_end_intersected_decoded_media`; no different clip or silent fixture fallback was substituted.
- Real matching accepted an `unknown` result: score 0.677446 against threshold 0.75, 7.405 seconds evidence, zero named mappings. This demonstrates actual model execution and conservative result handling, not successful identification or an independent biometric accuracy measurement. Controlled matching separately qualifies named-result assembly.
- [Separate acoustic measurements](evidence/processing-qualification.json): speech recognition edit distance 4/33 words (12.121% WER); dialogue 2/31 (6.452%). Speech diarization generated two voices/four turns; dialogue one voice/six turns despite two published dialogue characters. Counts/coverage are diagnostics; no independently verified tight boundaries or DER claim exists.
- Final measured diarization peak RSS: 1,215,209,472 bytes (speech), 2,716,884,992 bytes (dialogue). Receipts record per-operation elapsed/CPU time, two elected CPU threads, 120-second operation bound, offline models and bounded outputs. RSS is measured, not capped by a portable hard memory limit.
- Selected recognition versions: faster-whisper 1.2.1, CTranslate2 4.8.2; diarization pyannote.audio 4.0.7, torch 2.14.0+cu130, torchaudio 2.11.0+cu130. CPU election does not prove CUDA compatibility. Exact model revisions/file hashes are retained in the receipts.

## Major-version documentation audit

Rescanned all 23 current documentation pages and reviewed product/architecture, technology, media/import, transcript/speaker/model/pipeline, storage/catalog/graph/security, desktop, development, roadmap, contract, release and navigation vocabulary. The [raw scan](evidence/glossary-scan.json) preserves candidates, including nonconcept tokens, for review. The glossary now has 156 entries, each with a primary learning resource and local usage links.

Classifications used during review:

| Token family | Treatment |
| --- | --- |
| Technical concepts and acronyms | Existing definitions retained; added ABI, AES-GCM, Argon2id, CAS, CRUD/DDL, CUDA, metadata families, MIME, RMS, XDG, signing/notarization, text encodings, DNS, FIFO, HTTP, JVM, OCR, CA/CDN, GPL/LFS, timestamps, architectures, CI, CSV, HTML, OS, hashes, URL and CC BY. |
| Runtime/entity/API names | Explain through their declared local contract and linked underlying concepts (for example `MediaEntry`, `ApplyRevision`, `Materialize`, `QuerySpec`). These are identifiers, not additional general technologies. |
| Environment/configuration names | Retain exact spellings and local ownership/allowlist semantics (`APPDATA`, loader/device paths, deployment names and credential namespaces); cross-link credential, OS, CUDA, XDG and release definitions. |
| CLI placeholders and status/enum values | Treat `FILE`, `DIRECTORY`, `REVISION`, `SOURCE`, `SKIP`, HTTP verbs and similar tokens as syntax documented at use. Do not invent glossary concepts from examples. |
| License/standard/source labels | Define CC BY/GPL, link standards from the relevant term, retain upstream tool/model identities. Roadmap IDs and diagram directions are syntax, not public planning links. |

The current index and landing/docs pages identify an unpublished candidate. Baseline routes remain; advanced CLI-only maintenance and interface gaps retain their capability matrix. Frozen contributor/changelog references match current root files. Product publication and hosted documentation deployment remain distinct operations.

## Local verification

| Check | Result |
| --- | --- |
| `npm run check` | PASS; UTF-8/BOM/LF/mojibake, Markdown/navigation/JSON/YAML/kit checks and 23 schemas/21 contracts/28 examples. |
| `npm test` | PASS, 82 tests. |
| Python maintainer tests | PASS, 106 tests, including Windows native credential lifecycle, both review rounds' regressions and native/generic CLI lane checks. |
| Processing worker tests | PASS, 16 deterministic tests; no acoustic inference. |
| `go test ./...` | PASS, all packages. |
| `go vet ./...` | PASS. |
| Processing/contracts/runtime race checks | PASS. |
| Desktop frontend tests/type check | PASS, 73 tests and TypeScript check. |
| Site/public/offline export | PASS, 54 generated routes, 52 checked export pages in each export. |
| Baseline byte inventory | PASS, all 46 unchanged. |
| Windows native media/Cueson/LadybugDB/current restore | PASS, `scripts/qualify.py native`; no inference in required qualification. |

The earlier local source preparation attempt correctly refused an unconfigured MSYS2 source-build toolchain. Existing verified runtime media-tool bytes were used for elected local qualification; that attempt is not six-platform package/source-build evidence. Complete native package/source qualification and candidate assembly require CI below.

## External completion

Pending official PR publication, six native variants/candidate collection, final-head CI timing and review disposition. T014 and T020 remain open until their evidence arrives. No success is inferred from workflow text or synthetic archive tests.

2026-10-10: Published [PR #43](https://github.com/shruggietech/insonic/pull/43), attached it to the task and moved issue #42 to In review. The initial automatic Codex review produced two findings: publication archives retaining candidate text, and released homepage state/downloads falling back to specification. Implemented an elected release-target public/offline render, retained in the exact-source build marker, required before publication/retrieval/promotion. Ordinary PR artifacts remain candidate snapshots. Added explicit released homepage state and six exact-tag CLI/desktop links. Both actual local exports passed: release target has six links/no pending controls/a release index; default snapshot has no release links/pending controls/a candidate index. No publication or deployment occurred during these checks.

Also corrected future preparation when upstream Cueson and insonic share a version number: upstream fixture bytes and external tags remain unchanged. The regression failed before the fix. Initial CI and review disposition remain in progress; a second review has not yet been requested.

The first CI attempt built and retained the cold Windows source-media cache (405.969 seconds of FFmpeg compilation), exceeding the ten-minute overall turnaround target. Five native package variants passed; the Windows desktop job stopped before assembly when its pinned OpenSSL download timed out. Acquisition now retries transient network failures up to three times within a 60-second acquisition budget, preserves the exact checksum/size limit, installs only verified bytes atomically and removes partial downloads. Corrupt bytes and permanent HTTP failures are not retried. Dedicated tests cover recovery, identity rejection, repeated failures, cache reuse and exhausted budget. A complete cached candidate run remains required.

The authorized second review on `b9a4f79` produced one additional finding: current schema examples retained baseline archive names and incorrect Linux ZIP formats, plus baseline common-type description text. Preparation now derives owned package names/formats for the selected version and updates only the owned common-type introduction. Corrected the current two schemas while preserving the immutable baseline. The new preparation regression failed before correction, then passed with the full 102-test Python suite, repository/schema checks and actual public/offline contract rendering. Both first-round threads are answered/resolved. No third review will be requested.

[The first complete candidate run](https://github.com/shruggietech/insonic/actions/runs/38025395531) passed all six real packages, all platform native/credential/bridge checks and the full alternative-backend fixture matrix on branch head `b9a4f7994d0150bb9b399ddaa74ed171c669bf4b`. Its clean actual build revision is `ba69f6d0750dae8aa19a551189eac1de6f3c14d0`, attempt 1. [Retained candidate/CI receipt](evidence/candidate-ci-first.json) records all six source-complete package inventories/hashes, matching public/offline archives and artifact digest. Collection executed its actual selected-source verifier, rather than relying on fixture archives.

Execution took 617 seconds (10m17s), plus 93 seconds queued behind the cancelled failed run, for 710 seconds total. This exceeds the target. Native jobs restored the existing core Go cache primary key and explicitly skipped saving their tagged/native additions. Runtime/desktop now share a distinct dependency/frontend-bound native key, and CLI has its own key; core cache scope and every check remain. The new key must first be retained, then a complete warm run measured. This is a correction in progress, not an accepted timing waiver.

[Corrected-source candidate evidence](evidence/candidate-ci-warm.json) binds head `088244c3ee412a29bb3d998509ea5a051cb3d51f`, clean build revision `95be94d86e5d7425735566b5b5cc7bc0eb960959`, [run 38026179037 attempt 2](https://github.com/shruggietech/insonic/actions/runs/38026179037/attempts/2) and all six packages/documentation. Attempt 1 filled the native keys and passed collection, but Windows native Go compile/test hit its unchanged 180-second command bound. Windows desktop retained the complete native key; the unchanged full rerun passed every check in 646 seconds execution, 650 seconds attempt turnaround. That still exceeds the target.

The remaining generic CLI stage took 195 seconds and built a different compiler/graph variant from the native package. Native lanes now elect `qualify.py cli --native`, sharing the selected compiler/loader and packaged `system_ladybug` build; actual LadybugDB availability is required before the same CLI scenarios continue. Generic lightweight lanes still require no native assets. The actual Windows native CLI qualification passes locally; the 106-test Python suite covers lane selection, environment restoration on failure, native build identity and fallback rejection. No existing scenario or timeout was removed/extended. Final complete native-mode timing remains pending.

## Readiness and unexecuted behavior

Read-only repository inspection found no signing/deployment variables, no repository secret names and no documentation-deployment environment. Signing/notarization and hosted deployment are unconfigured, not qualified. No identities, hosts or credentials were provisioned. No product tag, release, deployment or owner merge was performed.

Real-engine measurements cover Windows CPU and the selected short licensed fixtures only. Linux/macOS acoustic quality, CUDA, other models/providers, broad speaker-identification accuracy and base-weight retraining were not measured. Native package acceptance remains a separate deterministic matrix. After squash merge, release assets require a new clean exact-main candidate.
