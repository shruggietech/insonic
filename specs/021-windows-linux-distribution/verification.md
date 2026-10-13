# S021 verification

## Local implementation checks

| Check | Result |
| --- | --- |
| Full Go tests and vet | Pass on Windows with the configured UCRT64 compiler and CGO enabled |
| Repository check | Pass, 1025 text files, 23 schemas, 21 contracts and 28 examples at this checkpoint |
| Repository JavaScript tests | 84 passed |
| Full Python regressions | 155 passed, including current/historical release validation and seven executed Windows ownership tests |
| Processing worker regressions | 17 passed, including unsupported-host rejection before input/engine election |
| Native decoder pin checks | Five passed |
| Frontend TypeScript and tests | Pass, 73 tests |
| Frontend production build | Pass |
| Public and offline help build/check | Pass, 78 pages, resolving relative links and anchors |
| Native media qualification | Pass, real import/extraction, format/precision/grouping/replacement and playback fixtures |
| Extracted Windows CLI/desktop | Pass for CLI (152.781 seconds) and desktop (221.703 seconds); complete helper copy, all five first-use commands, full CLI workflows and desktop native bridge/webview |

The original Windows helper reproduced recursive copying through a source-contained destination/junction and leaving a partial destination after failure. The replacement passes complete spaced/non-ASCII copy, existing/source/relative/descendant rejection, real copy failure, concurrent destination creation, junction/SUBST alias rejection and held-attempt substitution tests. Existing user files remain intact. Current candidates additionally require successful Windows installation/first-use receipts; 1.0.0 retains its historical requirements.

Package verification rejected a direct local media build without independent dependency-group receipts. The staged source-build path used by CI completed locally with compression/audio/AV1 group receipts, joined dependency receipt, codec continuation and final binary/source proof; no inventory, source or qualification requirement was relaxed. The pinned Meson 1.8.3 tool environment and isolated fixture profile paths are local prerequisites, not product behavior changes.

## Convergence and delivery

The local package receipts at build/native/package-cli-receipt.json and package-desktop-receipt.json bind base revision ae630a19f50f5ff2027435192fce681d7d07e9e8 with source_dirty=true. They are development-tree checks, not release artifacts or committed-head CI evidence. No inference/acoustic accuracy or public release is claimed. The initial headless PowerShell native-command invocation relied on PATHEXT. With that variable absent it opened fixture EXEs through Windows Terminal; the owner reported two visible launch-error windows. That launch pattern was stopped, and only the two matching fixture windows were closed. First-use CLI children now use explicit ProcessStartInfo with UseShellExecute=false, CreateNoWindow=true, redirected input/output/error, bounded waiting and the installed working directory. Both variants passed complete copy and all five first-use commands with PATHEXT deliberately absent; fixture cleanup completed and no fixture terminal remained. This supersedes relying on inherited executable discovery.

Spec Kit Converge assessed 14 functional requirements, five success criteria, eight acceptance scenarios, four plan decisions and five constitution principles against current source, contracts and executed local evidence. No remaining buildable gap was found, and convergence appended no task. All local implementation tasks are complete; publication/CI/review closure remains a delivery gate. Project 7 tracks #73/#63 as S021, In progress, milestone Post-release usability. External review rounds used: zero. PR publication, exact-head CI and review-thread closure remain pending. The owner retains final review and squash merge; no tag, release or documentation promotion is authorized.

2026-10-13: Initial external Code Review completed on 2e6a4410feeaf96b146ca9dccb4176850bd71e2b with a thumbs-up and no findings/threads. The owner-reported visible-window correction is validated locally and will receive the one permitted follow-up review request after push. No third round will be requested.
