# S062 verification

## 2026-10-31 local implementation and pre-publication checks

Scope: #202-#208, branch codex/062-local-processing-reliability. Product, schema and current documentation remain version 1.0.3; this is unreleased work. Owner explicitly authorized push and an official PR, at most two Codex review rounds total, and an owner-controlled final merge.

- Spec Kit specification, planning, research, data model, contracts, checklists, task generation and analysis completed. Requirements and contract checklists pass; initial analysis found no blockers.
- npm run check passed: foundation, governed brand bytes, links, encoding, 25 schemas, 22 contracts and 30 examples. npm test passed 221 tests after integration.
- Frontend TypeScript check and all 109 tests passed. Site tests passed 18 tests; final site build validated 169 rendered pages and links.
- Full affected Go suites passed for processing, voicemodels, app, pipeline, library, workers, catalog, artifact, contracts and CLI. go vet passed the five changed runtime/domain packages.
- Python worker, probe, native-loader and operator-validator regressions passed. Exact-silence/nonzero fixtures, finite fallback, cancellation, failed-attempt receipts, elected native-library boundaries and historical CPU behavior are covered deterministically.
- A built Windows candidate passed actual headless operator qualification, including normal work output, missing-work/help routes, artifact grace/reference refusals, exact identities and capture readback (88.171 seconds; shared preparation 2.469 seconds).
- Actual documented workflows passed model acquisition/show/verify, CUDA inspection input, preset/pipeline inputs, trainer integrity, retrieval and short waits (preparation 1.156 seconds; qualification 59.172 seconds; combined 60.328 seconds). Two harness/input-shape mistakes found during execution were corrected before publication.
- Internal independent review found and corrected swallowed CUDA embedding-probe errors, receipt-unbound readiness, and lost failed-fallback observations. Added regressions passed; these internal reviews do not consume external Codex review rounds.

The operator/documentation qualification was repeated successfully on frozen candidate SHA-256 80a2ec9b5866f79b9e65a0aaac1e6d35c4383b915b6eaa874bde74b806dfc5df, with all 701 runtime/CLI/Go/worker source files independently matched byte-for-byte to commit 7f8c4f25a68c79d3bedcbbc39e1e0f31b4736ded. This final run used 2.203 seconds preparation, 91.860 seconds qualification and 94.063 seconds combined time. Hosted source/package qualification remains required. Compact and private detailed receipts are retained outside workflow runs in the owned local S062 evidence directory.

## Pending delivery gates

Separately elected actual CPU/NVIDIA transcription, diarization, enrollment and matching acceptance passed all 13 rows on frozen candidate SHA-256 80a2ec9b5866f79b9e65a0aaac1e6d35c4383b915b6eaa874bde74b806dfc5df. The original qualification clock finished in 383.210 seconds (378.272 seconds prerequisite preparation, 761.482 seconds combined; compiler invocations 20.956 and 6.316 seconds reported separately). The 1,800-second target, 3,600-second hard limit and 60-second finalization reserve were preserved, including the harness-only correction interval. Terminal time: 2026-10-31T11:13:46Z. Owned runtime and descendants were released.

The retained [engine receipt](evidence/windows-engine-acceptance.json) covers omitted-auto CUDA, explicit CPU, exact silence on both devices, invalid-language digest/revision retention, CPU/CUDA enrollment and matching, and fresh GPU-hidden recording/enrollment/matching CPU fallback. Same-source heldout matching stayed unknown at the frozen threshold 0.75 (scores approximately 0.68613); no positive identification or broad acoustic-accuracy claim is made. The [operator/documentation receipt](evidence/windows-operator-docs.json) records its separate final candidate run. Both are compact copies retained outside workflow runs.

Windows/Linux source/package, supported backend fixtures and exact-head applicable hosted checks remain pending. Official PR #234 was published with initial automatic Codex review (round one) and CI run 41195652431. No separate security receipt exists yet. The earlier transient ArcadeDB observation remains an unconfirmed diagnostic case, with no claimed root cause. No merge, tag or release is authorized by this slice.

A subsequent internal check found default workers.inspect still discovering libraries outside its configured receipt. The read-only routing path now passes the matching recognition/diarization receipt, while explicit interpreter inspection retains discovery. Both-role regression tests and app vet passed. This follow-up is separate from the frozen inference candidate and will receive final-head hosted verification.

## Initial external review

Round one reviewed commit 7f8c4f25a68c79d3bedcbbc39e1e0f31b4736ded and found two actionable items: discover Linux system-linker CUDA paths within elected receipts, and retain the task-specific no-speech/no-timed-subtitles outcome in concise human output. Both are corrected. Linux system-linker discovery now records actual absolute mapped CUDA paths and bounded companion libraries, and frozen receipts suppress rediscovery. Model-free tests passed on actual Ubuntu 24.04 and Windows. Human work output retains a finite recording Outcome line, with golden and installed-harness coverage. Each comment will receive its own response before the authorized second round. No further review rounds are permitted.

## Consolidated local validation

The full Python suite passed 728 tests with four platform-specific skips using the existing repository-pinned Go 1.27.1 compiler. An earlier installer-source test detected the shell-selected older Go compiler; the launcher PATH was corrected, with no product or prerequisite weakening. Repository foundation/schema checks and all 221 repository tests passed again.

The distinct [follow-up candidate receipt](evidence/windows-followup.json) records receipt-bound configured inspection, explicit interpreter discovery, configured readiness and completed no-speech human readback (0.422 seconds preparation, 43.406 seconds qualification, 43.828 seconds combined; 3.352-second build separate). It introduces no new inference claim and preserves the original accepted engine bytes.

Initial hosted CI run 41195652431 completed with source/package/backend/docs/candidate jobs successful, but both core jobs failed the same outdated disconnect-fixture bootstrap matcher. The fixture now matches the canonical ten-argument launch and retains its original blocking, cancellation and cleanup assertions. Full local runtime tests and vet pass. Initial native/package receipts, job audit and failed aggregate acceptance are retained outside Actions before the next run. Final-head hosted acceptance remains required.

The [final operator receipt](evidence/windows-operator-followup.json) passed all media-enabled cases on the follow-up binary, including actual no-timed-subtitles assembly and human show/wait output (1.172 seconds preparation, 89.094 seconds qualification, 90.266 seconds combined). Candidate bytes remained unchanged.
