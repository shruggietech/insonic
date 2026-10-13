# Verification and handoff

## Scope and authority

S025 completes #69 through shared worker setup, builtin models and five-task readiness. Owner authorization includes automatic push/official PR and at most two external review rounds. First-run orchestration (#66), MSI (#65), installation screenshots (#64), tagging, release, deployment and merge are excluded. Software/docs/schema remain candidate 1.0.1.

## Local automated verification

- Root npm check passes file encoding, links/navigation, version/brand integrity and 25 schemas with 30 validated examples. Root tests: 86 passed.
- Go tests across all packages and vet passed; affected app/catalog/backup/model/worker/runtime/schema suites were repeated after convergence fixes.
- Frontend typecheck/production fixture build and all 80 tests passed, including rendered setup election, protected token registration, work cancellation and disabled stale-selection controls.
- Python qualification/tooling: 170 tests passed, including package metadata/hash-lock tampering and native protected credential restart.
- Public and offline documentation exports: 79 pages each, valid links/anchors. Historical 1.0.0 docs/schema are unchanged.
- Meaningful failing regressions preceded fixes for environment APIs, offline wheel filenames, portable local/setup source retention, protected Windows directory ownership and exact CLI help templates.

## Actual Windows CPU evidence outside CI

A fresh native CLI was built with system_ladybug and used from an owned temporary installation directory containing spaces, with verified package companions and current worker metadata. A separate disposable library exercised shared CLI/runtime operations without changing a user library or global Python/PATH. This is an installed-layout product journey; final official archive qualification belongs to exact-head CI.

Both managed roles downloaded declared Python/uv/wheel assets, performed offline hash-locked installation and published verified environment receipts. Recognition completed its observed wait in 12.707 seconds. Diarization completed after a bounded wait reported ongoing environment creation. The profiles are CPython 3.12.15, faster-whisper 1.2.1/CTranslate2 4.8.2/PyAV 18.1.0 and pyannote.audio 4.0.7 with torch 2.14.0+cpu, torchaudio 2.11.0+cpu and torchcodec 0.17.0+cpu.

A real immutable Whisper Tiny acquisition exposed the official us.aws.cdn.hf.co redirect, then resumed the same failed work after the narrowly scoped transport correction and completed in a 3.131-second observed wait. Community-1 reused an explicitly selected exact local snapshot, verifying all 32,821,421 bytes without network authentication or source mutation. Its acquisition completed.

Five-task readiness returned import, supplied subtitles, transcription and diarization prerequisites available. Training remained unavailable without an elected adapter/dataset. Recognition detected one GPU but correctly reported missing CUDA libraries; diarization reported CPU-only CUDA unavailability. No GPU fallback or inference was performed by readiness.

The real local CPU pipeline imported the committed speech fixture and successfully transcribed, diarized, assembled and published a validated current Cueson document. Its successful observed processing wait was 23.767 seconds. An earlier harness invocation deliberately retained a nonempty CI marker and hit engine_ci_forbidden; that guard remains unchanged. The outside-CI restart/retry completed with the original frozen models/device.

The opt-in qualify-processing.py harness then executed both licensed speech/dialogue fixtures with these actual managed interpreters, CPU, two threads, private offline child homes and the product builtin model pins. The sanitized cpu-qualification.json records exact hashes, models, versions, resource limits and measurements. Recognition wall times were 1.657/1.020 seconds, WER 4/33 and 2/31; diarization wall times 22.606/29.104 seconds, peak RSS about 0.90/2.38 GB and 4/6 turns. Dialogue produced one acoustic voice versus two published characters, retained as a diagnostic. These observations do not establish diarization accuracy or a portable hard RSS limit.

## Limits and external completion

Actual Linux model execution, CUDA execution, new-model upstream gated authentication and a locally attended native GUI journey were not performed. Deterministic fixtures cover credential transport/redaction, official CDN bounds, local linked snapshot reuse, frozen storage, portable recovery, cancellation/revision authority, changed bytes and package closure. Alternative-backend/native/package CI and every external review thread must pass on the final PR head before owner handoff. No separate security-bot verdict is inferred from a code-review result.

Publication and exact final completion-head checks are recorded in the official PR.

## Initial external review

Official PR #90 opened at 1ef0cdb27d7408f718099e48fa389ee451ebfef5. The automatic initial review returned two P2 findings: colocated filesystem peak capacity (r4262506536) and missing upstream credential references before gated enqueue (r4262506548). Both are fixed with regressions; affected Go tests/vet, frontend typecheck/80 tests and root integrity checks pass. Physical Windows volume identity/Linux device identity include mount/link resolution, previews do not create a missing destination, and unknown relationships use a conservative combined budget. Preview/local/available models remain usable without new authentication; only the corresponding elected gated download requires its protected reference.

## Final external review

The one authorized second-round request used @codex security review on 81ea8ad256f251b5ec3075a5519972d813b171ac. The service reported it as Code Review, completed at 2026-10-13 18:47 UTC; no separate security-service verdict is asserted. It returned two P2 findings: archive substitution between path verification/extraction (r4262667308) and managed-space requirements incorrectly applied to reuse (r4262667323). Both are fixed and regression-tested. Archives now copy/hash into an owned attempt before extraction, and reuse uses a 32 MiB receipt/probe budget in plan and execution. No third review will be requested.

## Final implementation and owner handoff

Final implementation head d1c174ca75942fa8401af693bd50a92bcc3b0dcd passes CI run [38450639383](https://github.com/shruggietech/insonic/actions/runs/38450639383): 18 successful checks and 14 intentional cached-stage/lane skips. Both platform core/native engine/desktop lanes, live PostgreSQL/S3/ArcadeDB fixtures, all four package variants and exact-source/attempt candidate integrity pass. Earlier complete runs also passed on 1ef0cdb and 81ea8ad. Completion records alone are committed subsequently; their exact-head checks and clean synchronization are verified in PR #90 before handoff.

The corrected native Windows CLI then performed actual offline managed recognition setup from the declared local archive/wheel bundle in a second disposable library. The successful wait was 9.626 seconds, with no elected network acquisition. Its compatible existing-interpreter preview reports exactly 33,554,432 required bytes and zero download bytes. This directly exercises the owned archive snapshots and corrected reuse budget.

Both authorized rounds are complete. All four findings are fixed, answered and resolved: initial capacity/access replies r4262612723/r4262613122 and final archive/reuse replies r4262755560/r4262756034. The complete review-thread query has no further pages and zero unresolved threads. No third review was requested. The security request was reported by the service as Code Review; no separate security-service verdict is asserted.

All 33 tasks are complete. PR #90 remains open for owner final review and squash merge with branch deletion. No merge, tag, product release or deployment is performed. The software/docs/schema version remains candidate 1.0.1.
