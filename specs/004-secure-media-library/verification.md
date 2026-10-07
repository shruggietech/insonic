# S004 verification and review ledger

## Identity and scope

S004, `specs/004-secure-media-library/`, branch
`codex/004-secure-media-library`, issues #5 and #6. Base main:
`b585ff33123588f0b3eae12bd75eb86229608df8`. Issue #21 timing/current-result
requirements are applied without claiming its full Cueson assembly is complete.

## Spec Kit sequence

The installed specify, clarify, checklist, plan, tasks and analyze instructions
were executed before implementation. Feature/setup/prerequisite helpers resolve
this exact feature. No extension hooks are registered. Analyze has no unresolved
critical scope/design decisions. Tests were introduced before their story
implementations, including native/vault, historical migrations, source/timing,
manifest mismatch and real-work paths.

Implementation/convergence found and resolved bounds preservation, live credential
selection/authentication refresh, restored manifest byte identity, reconciled
download cleanup, complete extractor support-tree sizing, and large batch
journal/IPC paging. Appended T029-T033 trace those findings. Additional fixes bind
cleanup to accepted replacement receipts, avoid catalog-wide metadata reads during
work recovery, and use one workspace-scoped SQL retirement reference query.

## Local executed evidence (Windows amd64)

- `npm run check`: repository text/link/navigation/brand and 17-schema,
  15-contract registry checks passed; no BOM, CRLF or mojibake findings.
- `npm test`: 51 maintainer tests passed after acquisition-contract additions.
- `go test ./...` and `go vet ./...`: passed.
- Affected app/library/model/secrets/credentialcmd/runtime/desktop race checks:
  passed. Catalog/artifact authority and retirement regressions passed.
- Nine Python qualification tests passed.
- Real SQLite and pinned Ladybug native transactions passed.
- Cross-process CLI and desktop bridge/offline help qualification passed.
- Native Windows credential add/status/resolve/replace/delete and separate-process
  restart passed; responses contain no credential values.
- Exact packaged ExifTool 13.59 and ffprobe
  `6.1.1-essentials_build-www.gyan.dev`: actual original import, 10 ms / 48 kHz
  timing, source retention and physical refresh retirement passed. Archive and
  delegated support-file digests are locked with retained licensing notices.
- Public and offline documentation builds passed, including all 27 documentation
  pages and their relative assets/anchors.
- Genuine legacy v1/v2 catalog upgrades, model download interruption/resume,
  cancellation/fencing, exact digest/size mismatch, materialization, snapshot
  identity, 10,000-item journal/results, partial metadata/date states, reference
  relocation and current-result replacement have focused passing regressions.

## Hosted acceptance

CI runs Linux, Windows and macOS native credential lifecycle and exact extractor
operations. Linux uses a private unlocked Secret Service fixture; macOS uses an
isolated temporary native keychain. Adapter acceptance includes PostgreSQL and
generic S3 plus full library operations over all four catalog/storage combinations.
Initial head `76f9733f3fe50745903c7ce4d84ffefd2d0f3ab8`, CI
[run 37672587418](https://github.com/shruggietech/insonic/actions/runs/37672587418):
all six gates passed. Foundation 10 s, docs 31 s, adapter fixtures 1 min 46 s,
macOS 3 min 15 s, Linux 5 min 24 s, Windows 7 min 52 s. Real platform/backend
acceptance is executed, not inferred from compilation. Follow-up code must pass
the same hosted gates before final handoff.

The macOS job uses `macos-15` arm64 following GitHub's
[macOS 14 retirement announcement](https://github.blog/changelog/2026-10-01-github-actions-macos-14-runner-image-retirement/).
The extractor/native lock already targets arm64, so no artifact identity changes.

## Review protocol

Publication is explicitly authorized. Publish an official PR closing #5/#6,
attach it to the chat, then resolve every review discussion/comment. Review round
count starts with one automatic round. One `@Codex` second-round request is
authorized; never request a third round. No merge or release is authorized.

Official PR [#22](https://github.com/shruggietech/insonic/pull/22) published at
`76f9733f3fe50745903c7ce4d84ffefd2d0f3ab8`. Initial automatic Codex review
completed with four findings. T041-T044 fix unused source/subtitle candidates,
logical model-version conflicts, bounded large ingress and desktop deadlines.
T045 retains failed per-item results in retryable batch work without duplicating
admitted items. Real-runtime input larger than 1 MiB reaches durable work;
responses retain the smaller budget. All four have targeted passing regressions.

Independent review T034-T040 resolved current snapshot proof/omission, cold
credential bootstrap, semantic date/fold selection, CLI bounds, external native
deletion, malformed URL/offset validation and transient retirement recovery.
Latest receipt validation includes cleanup obligations and returned Work journal
identity. Local full tests/vet and affected race regressions pass after integration.
Repository check and 50 maintainer tests pass. Documentation explains established
PostgreSQL session lifetime separately from new credential resolution.

Initial four findings were fixed in `92ef53120058abf59f1784b463f6aac7cbcf523e`,
each replied to and resolved. Follow-up review completed on that head with three
security findings. T046-T048 address transport before secret resolution and
persistence, configured acquisition byte/time limits, and common JWT/session
aliases. CLI and schema options preserve batch/per-item inheritance. Boundary,
TLS, cancellation/deadline, partial-file cleanup and actual no-persistence tests
pass. Ordinary noncredential selectors and remote HTTP remain supported.

### Review-cap deviation

The initial automatic review was followed by the combined code/security request
[6045341738](https://github.com/shruggietech/insonic/pull/22#issuecomment-6045341738).
Only a code-review job appeared, so an additional security request
[6045363604](https://github.com/shruggietech/insonic/pull/22#issuecomment-6045363604)
was sent. The bot again displayed a code-review job. That extra manual request
exceeded the owner's request cap and was reported to the owner immediately.
It was not authorized as a third round; no further request will be sent.
Two completed bot review reports are visible (initial and follow-up), with seven
findings total. This ledger does not claim compliance with the review cap.
Final replies/resolution and exact-head CI remain pending.

## Remaining product boundaries

Transforms, inference, complete Cueson assembly, trained speaker-model creation,
full GUI controls and installable release packages remain their separate issues.
Configured admission reports no fabricated downstream job IDs. This work does
not claim qualification of every external S3/PostgreSQL deployment or physical
power-loss behavior beyond the executed fixtures.
