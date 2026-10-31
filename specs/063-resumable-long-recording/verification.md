# S063 verification

## Scope and acceptance

This slice implements issues #212, #213, #214, #215 and #216. Epic #209 and follow-on coverage/repair/qualification issues #217-#219 remain open. Push and official PR publication are authorized. Merge, tags and release publication remain owner-controlled.

## Requirement evidence

| Requirements | Implementation and meaningful checks |
| --- | --- |
| FR-001, FR-002; SC-001 | Versioned processing plan and window schemas, deterministic native-sample coverage, exact rational offsets, selected ranges/channels, uneven tracks, empty/final windows and overflow tests in `internal/processing/windows_test.go`. |
| FR-003 | Frozen source/document/configuration/model/tool identity, portable path rebinding with byte identity retained, changed-track implicit reuse rejection, and per-cue/per-voice source routing. |
| FR-004, FR-005; SC-002 | Hidden supervised stream decoder, bounded source-chunk verification, continuous resampling, active-window scratch, MP3 Xing/LAME gapless correction. Native full-decode byte oracles cover FLAC, MP3 and unequal-offset MKA; tenfold duration preserves live scratch bounds. Filesystem and S3 adapter fixtures cover source mutation, three source passes across twelve windows, cancellation and lease/scratch cleanup. |
| FR-006, FR-007; SC-003 | Native word evidence validation; temporal and lexical duplicate matching; genuine repeats, multilingual text, absent/ambiguous word evidence, simultaneous independent tracks and changed-output digest tests. Scoped Cueson assembly prevents cross-track voice assignment. |
| FR-008, FR-009; SC-004 | Catalog schema 11, typed plan/checkpoint records, generation fences, publication kind/hash/size checks, forged proof and partial-output rejection. Runtime tests cover interrupted retry, cancellation plus restart, corrupt retained bytes, unchanged current document and orphan publication cleanup. |
| FR-010, FR-013 | Ready-only backup artifact selection, restored checkpoint identity, stale authority expiry, retirement after current acceptance/source replacement, shared SQLite/PostgreSQL domain contracts, and existing artifact/graph authority. PostgreSQL integration tests are registered for CI; no local PostgreSQL fixture was elected. |
| FR-011, FR-012; SC-005 | Finite CUDA batch reduction and CPU fallback under one deadline; explicit CPU remains CPU; classified CPU memory exhaustion; stage/device observations survive checkpoint replay and retirement. Worker tests use deterministic fixtures. |
| FR-014; SC-006 | CLI window progress and help, saved pipeline/runtime schemas, generated public schema reference, authoritative pipeline/architecture documentation and changelog. |

## Local checks

Receipts are retained outside Actions under `.local/s063/`. Their original timestamps and stdout/stderr remain authoritative. These are separate command runs, not a claimed aggregate release qualification campaign; preparation, qualification and combined timing must not be inferred by summing parallel durations.

| Check | Observed result | Retained receipt |
| --- | --- | --- |
| `npm run check` | Passed, 27 schemas, 24 document contracts, 32 examples | `foundation-final.json` |
| `npm test` | 222 passed | `node-tests-converged.json` |
| `go test ./... -timeout 30m` | Initial full pass 164.622 seconds; final settled-tree pass 73.645 seconds | `go-tests.json`, `go-tests-accepted.json` |
| `go vet ./...` | Passed | `go-vet-final.json` |
| Python processing worker tests | 26 passed | `worker-tests.json` |
| Documentation production build and export checks | Passed, 169 exported pages with links/anchors checked | `site-build.json` |
| Desktop frontend type check | Passed | `frontend-check.json` |
| Native process/processing tests with pinned FFmpeg | Passed, 26.391 seconds | `window-native-qualification.json` and `.log` |
| Native Cueson with injected inference, positive origins and simultaneous source tracks | Passed; exact multilingual cues and independent assignments | `worker-stitch-qualification.json` |
| Final affected app, CLI and subtitle suites | Passed after source/reuse/progress changes | `app-final.json` |

The first Node suite exposed unclassified timing owners. The maintained inventory now classifies existing product cleanup/child-exit clocks separately from test fixture allowances; no product deadline was expanded. The first S3 fixture attempt exposed an unclosed replaced fixture store; that diagnostic attempt is retained alongside the corrected native pass.

## Convergence

The first convergence assessment checked 14 functional requirements, six success criteria, eleven user-story acceptance scenarios, eight plan decisions and the governing project constraints. Three high-severity partial gaps were appended as T031-T033: source-clock endpoint authority, implicit current-transcript reuse compatibility and publication-before-checkpoint orphan recovery. A portable Cueson executable-path identity regression was also corrected under FR-003. T031-T033 are implemented with regression coverage. Aggregate rendering now separates recording-wide counts from individual worker limits and reports output capacity as `output_limit`; per-window recognition/diarization disagreement remains explicit.

Final code convergence found no remaining implementation gap against the 14 functional requirements, six success criteria and eleven acceptance scenarios. Required local checks pass. The settled-tree full run exposed an existing 10 ms assistance cancellation test racing dependency preparation; it now cancels deterministically inside the elected fixture without changing a product timeout. The failing run is retained in `go-tests-final.json`, followed by the accepted full-suite receipt. Publication and project mapping are complete in PR #235. Exact-head CI and final review closeout remain T030.

## Limits of the evidence

Native FFmpeg extraction and native Cueson acceptance are separate from model inference. No actual CPU or NVIDIA recognition/diarization campaign was elected for this slice. Deterministic worker and runtime fixtures establish state, timing, routing, recovery and receipt behavior; they do not establish acoustic accuracy or hardware throughput. Local PostgreSQL was unavailable because `INSONIC_FIXTURE_POSTGRES` was unset; CI must supply the registered disposable fixture. No release assets were built or published, and retained earlier-slice release evidence was not modified.

## External review and CI convergence

The initial Codex review identified two defects. Supplied admission transcripts now retain their independent native document clock (including an absent clock) while source identity and acoustic reuse remain fenced. An explicit audio stream that is unavailable now returns `unsupported_audio` instead of falling back to the first audio stream. Regressions cover the saved admission pipeline, late supplied cues, native Cueson validation and rejected cross-track diarization reuse.

The first CI attempt exposed shared Go files accidentally carrying the Windows platform suffix and an installed-contract check requiring catalog schema exactly 10. Shared window files now use platform-neutral names; the installed duration/Unicode check requires schema 10 or newer and independently verifies the persisted values. The actual schema-11 CLI passes those duration and Unicode boundary cases. Windows affected-package suites and native Linux window/reuse regressions pass. Linux execution used verified local Go module archives because the installed WSL network was unavailable.

Run 41221606062 is retained as a failed diagnostic attempt. Its digest-verified acceptance artifact and terminal audit are outside Actions in `.local/s063/initial-ci/`; the audit records 175 seconds preparation, 687 seconds qualification and 862 seconds combined. Functional failures keep that attempt rejected. Final branch-head qualification and the second review are pending; no third review request is authorized.
