# Implementation Plan: Windows desktop lifecycle repair

**Branch**: `codex/040-webview-lifecycle-repair` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

## Summary

Deconstruct #154 across the application, Wails, the maintained native binding and the Windows/browser ownership boundary. Start from the exact merged product tree. Fix the independently reproduced native BOOL ABI error that upstream corrected in go-webview2 v1.0.23. Retain the existing lifecycle fixes while adopting the authoritative upstream source. Add only the observations needed to distinguish final response completion and release timing if those boundaries remain unresolved. Native qualification follows a controlled experiment, never speculative full-package retries.

## Technical Context

**Language/Version**: Go 1.27.1; Python 3.12; Node 22+; existing TypeScript frontend.
**Primary Dependencies**: Wails v2.16.0, maintained go-webview2 v1.0.22 moving to authoritative v1.0.23 with prior owned-lifetime patches retained, Windows COM/User32, installed WebView2 runtime.
**Storage**: Existing local/remote backend contracts unchanged; disposable profile/evidence files only.
**Testing**: Actual native fake-COM vtable regressions with message-only or unshown windows; normal and race Go tests; Python process/harness tests; Node source-authority tests; root and site checks; disposable hosted native source and package qualification.
**Target Platform**: Repair targets Windows x86-64; Linux compilation and existing desktop behavior remain required.
**Project Type**: CLI-first desktop application with a shared runtime and versioned source/package qualification.
**Performance Goals**: Original per-case deadlines, bounded diagnostic sampling and complete configured CI within 660 seconds.
**Constraints**: No owner-desktop GUI/input, no ambient process targeting, no content dumps, no speculative timeout increases or weakened success gates. No third review on PR #156, no future merge/release authorization inferred.
**Scale/Scope**: Issue #154 only, with five retained signature classes. Historical S039 remains completed under its explicit exception.

## Constitution Check

| Principle | Design assessment |
| --- | --- |
| I. Owner requirements | Full repair scope retained; unresolved evidence remains unresolved. |
| II. CLI-first/shared behavior | No core operation or backend change; desktop teardown preserves shared work. |
| III. Source identity | Workspace/media untouched; all runs and modules retain exact identity. |
| IV. Adapters/credentials | No routing/configuration change; content-free bounded observation. |
| V. Efficient/truthful work | Reproduce the cheapest known violation first, then one discriminating native experiment; never claim finite passes prove a rare issue absent. |

Pre-research and post-design checks pass. The unknown private runtime cause is an investigation outcome, not an unresolved product requirement. Failure to demonstrate repair keeps tasks and #154 open.

## Project Structure

- `specs/040-webview-lifecycle-repair/`: specification, research, responsibility map, data model, contracts, tasks, experiments and verification.
- `third_party/go-webview2/`, source manifest, `go.mod`/`go.sum`: upstream BOOL fix and checksum-bound maintained lifetime patches.
- `scripts/webview-source.mjs`, corresponding Python packaging authority and source tests: version/patch integrity.
- `cmd/insonic-desktop/`: bounded startup/release stage observations only where evidence requires them; native actions/deadlines unchanged.
- `internal/process/`, `scripts/process_tree.py`: owned-process proof and timing if required by the release signature.
- `.github/workflows/installer-invocation-diagnostic.yml`, `scripts/diagnose-desktop-startup.py`: reuse the single targeted disposable source experiment, no packaging diagnostic loop.
- `CHANGELOG.md` and affected current documentation: final demonstrated behavior and dated dependency decision, no numbered planning links in public docs.

## Chronological implementation sequence

1. Verify squash identity/tree equality, fast-forward main, remove verified merged local branches, close completed milestone and verify project automation. Completed before S040 branch creation.
2. Specify/clarify/checklist/plan/tasks/analyze S040. Installed Spec Kit Python helpers resolve templates and feature state; no extension hooks exist. Phase 0 research agents independently audit responses, native lifetime and harness/environment under the installed plan skill.
3. Add actual native vtable regressions for false/true argument values in the actual production Controller3 wrapper (the unused generated API adopts verified upstream bytes without execution qualification). Run against current v1.0.22 and retain failures before changing dependency source.
4. Acquire v1.0.23 through verified Go module authority, compare its two-file upstream delta, retain original maintained lifetime modifications and update exact checksums/allowlists/notices/source packaging. Repeat the same regressions and existing native ownership suite.
5. Complete the source responsibility map and reject disproven CSP/content-length/profile/input assumptions. Instrument only unresolved native response completion, actual runtime identity and end-to-end release timing, behind existing disposable qualification observation. Tests must preserve native calls, errors, ordering and disabled behavior.
6. Run one written targeted source experiment, stopping at first failure or its fixed budget. Classify the first incomplete boundary. If another experiment is necessary, it must answer a new discriminating question supported by the previous result and be recorded before dispatch.
7. Repair the demonstrated remaining violation under a failing regression, then run cold/warm, with/without-library and all actual native exits, followed by final ordinary CI/package gates. A failure is not cleared by later passes or cleanup.
8. Converge task/evidence state, update #154 and current documentation/changelog, and provide a truthful reviewable repair handoff. No merge or release occurs.

## Decisions and alternatives (2026-10-19)

- Adopt the exact upstream v1.0.23 correction instead of retaining a known wrong ABI or broadly upgrading Wails. The upstream comparison changes only two wrappers, and the product executes the affected edge method during controller initialization. Native fake-vtable tests distinguish the erroneous pointer from canonical FALSE/TRUE. This is a demonstrated startup contract violation; full incident causality still requires native evidence.
- Keep the legacy callback QueryInterface and additional interface-retention findings independent. They are real source concerns, but mass callback migration before assessing the directly relevant upstream fix would introduce a second behavior variable.
- Prefer edge-boundary observation over a full Wails fork. A returned application handler precedes HTML rewriting and native response completion. Successful response/Complete observations move the investigation into browser consumption rather than inviting another host-side guess.
- Preserve actual runtime identity separately from available-version lookup. The pinned Go loader clears environment overrides; any runtime comparison must pass the framework's explicit browser path and verify the created environment version.
- Preserve one 18-second end-to-end Exit budget. Add stage timings if needed; do not mislabel a small remaining release budget as evidence of an indefinitely stuck browser.
- Custom checklist markers remain reviewer-owned and unchecked. Autopilot proceeds with the requirement-quality assessment from analyze while preserving those markers; implementation does not change them. The built-in requirements checklist is 15/15.

## Complexity Tracking

No architecture expansion or constitution exception is elected. A maintained SDK patch remains checksum-bound and distributed with corresponding source. Any later new framework copy or runtime selection must first be justified against the observed failure boundary.

### Pre-dispatch refinement (2026-10-19)

E002 uses a checksum-bound historical one-file Go overlay as a diagnostic control beside the corrected build. Each arm has its own150-second/120-child bound and first-failure stop; total sampling remains300seconds/240children. Payload layout and executable basename are retained. No source mutation or package qualification uses the control. This comparison adds causal information within the planned one-job budget; source/observer behavior, native deadlines and completion gates remain unchanged.

### Post-comparison refinement (2026-10-19)

E002 completed120historical and120corrected idle launches without failure; no incident-cause attribution is established. E003 restores the elected historical concurrent CLI contract on the same immutable runtime source, with its prewritten bounds and overlap/confound limits in experiments.md. Independent actual-vtable/GC failures now justify a focused9callback adoption of the existing combridge. Lazy creator ownership, canonical COM identity, balanced native retention and late completion/event rejection are required; the unused generated API, unrelated settings queries and runtime selection remain outside this correction. This follows the initial ABI comparison rather than mixing the two behavior variables.
