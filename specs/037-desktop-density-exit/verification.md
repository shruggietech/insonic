# S037 verification

Scope: #112 and #113 on codex/037-desktop-density-exit. PR #153 remains open for owner review; merge, release, tags and site publication are outside the authorized boundary.

## Planning and implementation gates

The literal issue/branch/directory mapping was verified before implementation. The built-in requirements checklist passed all 12 items. Initial read-only analysis covered nine functional requirements and five success criteria; a wording mismatch was corrected to name Setup as the sixth workflow group, and reanalysis found zero gaps. No custom reviewer-owned checklist or additional approval gate was introduced.

The allowed consuming-host adaptation uses the delivered interface, spacing and typography tokens. Governed kit bytes remain unchanged. Native checkboxes retain associated label targets of at least 44 by 44 while the glyph remains square. Ordinary actions use intrinsic widths; wrapped navigation and tables keep contained scrolling. Native File > Exit uses the ordinary Wails lifecycle through a synchronized startup/quit controller. It adds no runtime stop command, job cancellation, density preference or schema change.

## Local and renderer evidence

- The actual-renderer regression first observed a 995.953125 by 44 checkbox. The corrected shared component measures 16 by 16 inside a 169.046875 by 44 label target. Label-padding activation, Space and visible focus pass; deliberately stretched glyphs and short actions are rejected. Wrapped action assertions measure intrinsic text width capped by available space.
- Source e45f90a653314ff61bd119f77f2f5143d2eab2f9 passed all 96 mounted scenarios: six workflows, 1200 by 800 and 720 by 600, renderer scales 1/1.25/1.5/2, and normal/200% text. Every scenario passes actual Tab/Space focus and label-padding activation with values restored. Capture/build/check took 20.047 seconds.
- Seven density images, four setup landmarks and both original receipts were copied unchanged into docs/screenshots/v1.0.1. Receipts bind clean source, binary identity, image hashes, dimensions and complete browser/helper/fixture cleanup. Historical capture identity is preserved; it is not rewritten as the current head.
- Required repository check/test, all 349 Python maintainer tests, all 104 frontend tests/type check, affected desktop race/native-command tests and site check/test/build pass. UTF-8 without BOM, LF, mojibake and Git whitespace checks pass. The Windows input ABI and invalid/foreign-window guards are covered without sending input locally.

## Native ownership proof and diagnosed corrections

A separate fixture owner hosts the real shared runtime and a held synthetic model acquisition. Child GUIs use the production application options, native menu, bridge and shutdown path. After each GUI exits, direct IPC requires the exact original runtime session and durable work identities; it does not use an auto-starting client. Real ranged playback must have its ticket, listener and runtime handle released. Releasing the byte barrier must complete both acquisition and dependent work. Only the fixture owner cancels its own runtime after completion.

Chronological qualification corrections:

1. Both source platforms exposed a test-only relative companion path because Go tests run from their package directory. An explicit repository-root media-tools path fixes the fixture; the override-free local regression passes.
2. Windows exposed an incorrect internal form-class assumption. Selection now uses the exact owning PID, application title and actual native menu, with ambiguity rejection.
3. The hidden process STARTUPINFO can suppress Wails' first ShowWindow call. The explicit disposable child requests normal Wails WindowShow and waits for visible/enabled ownership before menu messages. Production startup remains unchanged.
4. Menu Exit and ordinary close then passed on Windows; the remaining Alt+F4 probe used posted key messages, which do not exercise the system input path. A discarded local invisible-window experiment confirmed that modifier fields alone were insufficient. The final probe uses SendInput only for its own visible, enabled foreground fixture window. Native menu-mode/owner assertions and all watchdogs remain mandatory; no callback or forced-process fallback can pass.

Primary platform references: [ShowWindow](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-showwindow) and [Windows input guidance](https://devblogs.microsoft.com/oldnewthing/20250319-00/?p=110979).

Both Windows and Linux source and extracted-package proofs now pass, including compact native workflows, no-workspace/workspace Exit, playback cleanup and exact work survival through successful completion. Windows additionally passes real Alt+F4 input; Linux activates the actual GTK menu item on its main loop. Existing installer, setup, recovery, cancellation, relocation and candidate-integrity gates remain enabled and pass.

## CI timing, reviews and convergence

Implementation head 0ae962a6ae42440bd43ea779be6edbe2a6455bc5 passed 25 checks with 18 intentional skips, but measured 669 seconds from scope start through candidate cleanup. This exceeded the 660-second criterion and produced convergence task T016.

Desktop cache identity now includes root/frontend/site dependency locks. CI elects collect --summary only after unchanged complete candidate verification. The default output and full manifest/artifact inventories remain intact. Regression coverage checks full-manifest equivalence, digest identity and corrupt-asset rejection in summary mode.

Head e7fec63f2027cf74b2cf072de1a664f22e4544eb passes all 25 checks with 18 intentional skips. Measured scope start 2026-10-18T00:17:53.1063640Z to final cleanup 2026-10-18T00:28:38.9050352Z is 646 seconds, within 660. The candidate summary reports four packages and eight assets with the complete release-manifest digest.

| Round | Review | Head | Result |
| --- | --- | --- | --- |
| 1 | Automatic code review | b0cdad3f6f8da184c2036ad956372201dd3a15e7 | Completed, no findings |
| 2 | Security review | f415a08241bc90878e92342ed5a85bb295f058d3 | Completed, no findings |

Direct review-thread queries report no unresolved threads. Later changes are qualification, CI-overhead and evidence corrections; no third review is requested. Issues and PR are In review in Project 7, slice S037, milestone 21.

Final read-only convergence checks nine functional requirements, five success criteria, eight acceptance scenarios, nine plan decisions and all five constitution principles. Missing/partial/contradicting/unrequested implementation gaps: 0. The assessment leaves tasks unchanged; delivery bookkeeping afterward marks the completed tasks. Final documentation-commit CI must pass before handoff, with its exact head/verdict recorded in the PR.

## Evidence limits and handoff

Renderer scale is not claimed as physical OS display-scale testing. Automated Windows/Linux CI and controlled Chromium observations do not certify every installed machine, assistive technology or OS/browser combination. The synthetic acquisition proves durable ownership and completion, not inference quality or a real model execution. No owner-attended pre-release gate is imposed.

Stop with the PR open, clean and unmerged for the owner's final review and merge.
