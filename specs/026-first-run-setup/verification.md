# S026 verification

2026-10-13, implementation before publication. S026/#66 matches `codex/026-first-run-setup` and `specs/026-first-run-setup/`. Spec Kit specify, clarify, checklist, plan, tasks, analyze and implement completed; no extension hooks installed. Explicit owner authorization covers push and official PR publication, at most two external review rounds, then owner merge handoff.

## Local checks

- npm run check: project text/link/version/JSON/YAML/brand integrity and 25 schemas/22 contracts/30 examples passed.
- npm test: 86 repository tests passed.
- go test ./... and go vet ./... passed. Later bootstrap/runtime/CLI/native/schema focused regression runs passed after final integration refinements.
- Python unittest: 170 tests passed; package-desktop focused suite: 19 passed after setup acceptance was added.
- Frontend typecheck/build and 97 tests passed, including 17 setup journeys (restart, lost responses, conflicts, protected values, mode persistence, keyboard focus, immutable retry, late workspace results and existing-library resume).
- Public/offline exports: 80 pages each passed.
- Nine runtime setup tests passed with the race detector; concurrent private bootstrap writer tests passed. The final frontend pass additionally releases a stale initial read after acceptance and verifies ordinary import routing remains explicit.

## Actual Windows CLI journey

A current `system_ladybug` CLI and exact-hash companion manifest ran in a disposable installed layout with spaces and an isolated profile/library. Location preview created no workspace. A saved local-default draft replayed with identical normalized workspace/profile identity; explicit provisioning succeeded. Shared setup state save/replay and settings alias reopening preserved deferred progress and the original accepted revision.

Five-task readiness reported import and existing subtitles usable while recognition/diarization/training dependencies needed attention. The real speech.flac/speech.srt fixture then imported successfully with current validated Cueson output and no elected pipeline, account registration or model download. Work succeeded, not merely enqueued. Source fixtures remained unchanged.

## Package/source qualification boundary

Complete local package assembly reached the configured owned-source media cache check and stopped because this host lacks the selected MSYS2 toolchain. No integrity check was bypassed. The disposable CLI journey verifies current runtime behavior and exact companion hashes; it does not establish complete source-closure package qualification. Existing CI native/package lanes have the configured toolchain and now exercise setup save/replay/reopen/readiness plus ordinary import after deferral. Their exact-head results remain pending until publication.

## Unverified behavior

No attended native GUI, screen-reader or increased-scale Windows session was performed. Automated UI keyboard/focus and layout controls were checked. Linux native execution, live supported alternatives and complete four-package/source closure await CI. No new model inference, GPU execution, installer/MSI, product tag/release, deployment or merge is claimed.

## External review accounting

Initial automatic review: pending. Authorized second request: not yet used. No third review will be triggered. Every review comment/thread and final exact-head CI will be read directly before owner handoff.

2026-10-13: Initial CI 38473384085 exposed Go 1.27.1 temporary-fixture directories with POSIX mode 0755. Production's private-directory guard correctly rejected them. Bootstrap tests now create their own 0700 configuration directory and add an explicit nonprivate-directory rejection case; the guard is unchanged. Offline Ubuntu 24.04 reproduction used the exact cached Go 1.27.1 toolchain and verified module cache. Affected bootstrap/runtime/CLI/native-bridge/schema tests then passed on Ubuntu and Windows. Native Linux packages remain a separate CI claim.
