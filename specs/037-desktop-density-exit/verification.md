# S037 verification

## 2026-10-17: specification, research and analysis

Owner-approved scope is #112 and #113 on codex/037-desktop-density-exit, based on main5d429dd67f320a813b6054797c78df104491ee42. No extension hooks exist. Built-in requirements checklist12/12 passes as a requirements-quality assessment, not implementation proof. No unresolved owner question remains.

Read-only planning agents examined the pinned brand extension guidance and Wails menu/lifecycle. The primary independently read the cited bundled guidance. Consumer compact composition retains delivered tokens and44-unit label targets without kit edits. A queued startup guard handles asynchronous lifecycle initialization. Native Exit proof must use a separately owned runtime rather than the self-cancelling existing webview smoke.

The read-only analyze gate found one HIGH wording mismatch: six workflow groups were claimed but five named. The follow-up specification clarification explicitly includes Setup as the sixth shared-control group; actual recording processing remains within Library native observations. Re-analysis finds zero unresolved findings, zero constitution conflicts, zero unmapped tasks and100% coverage of9 functional requirements plus5 success criteria by14 tasks. FR-001..005/SC-001..003 map toT004..007; FR-006..007/SC-004 toT008..011; FR-008 toT004/T006/T007/T010/T011/T013; FR-009/SC-005 toT012..014. The implementation gate is ready. Existing ignore patterns cover build/dependency/native outputs; no additional publish ignore or hook files are needed.

Product tests and native behavior remain unexecuted at this planning checkpoint. Owner authorization already covers push, PR and two rounds; stop before merge.

## Implementation evidence

- The focused actual-renderer regression initially observed a 995.953125 by44 checkbox. After consumer composition, the glyph measures16 by16 and its associated label169.046875 by44; label-padding activation, Space and visible focus pass. A deliberately stretched glyph is rejected by the shared observer.
- All104 frontend tests and the frontend type check pass. Menu composition and concurrent early/repeated quit requests pass under the Go race detector. Full desktop package tests and native command compilation/tests pass locally without launching a window.
- Read-only runtime ownership research found no existing short-idle CLI or stop command. The proportional implementation uses a separate Go fixture owner and real shared runtime outside the child GUI process, then direct IPC for survival assertions. This replaces the planned additional Python driver and avoids adding a product command or duplicating process ownership machinery. Native menu activation remains confined to disposable CI.

- Full renderer preflight identified a legitimate wrapped Settings label at720 by600 with200% text. The observer now measures the unwrapped intrinsic text width capped by available container width, rather than mistaking a wrapped line Range for its maximum. Deliberately stretched short actions still fail the regression.
- The actual media/acquisition owner fixture passes locally without Wails: current media revisions are read through IPC, real ranged playback is opened/released, held model work preserves exact session/identity and finishes after barrier release. This verifies fixture logic only; native activation remains a separate mandatory CI proof.

- Clean source5c370d7c13eac6bcfe32700df2236bd4a9de699f passed the complete96-observation renderer matrix and retained seven density images plus the four setup landmarks. Capture/build/check completed in15.735seconds with source/binary identity and complete browser/helper/fixture cleanup. Original receipts and PNG bytes were copied unchanged to the versioned screenshot directory. Renderer scale is explicitly distinct from OS display scaling.

- Required repository check/test, all348 Python maintainer tests, all104 frontend tests/type check, desktop race/native command tests, and site check/test/build pass locally. The native gate test retains the previous bridge/help/journey probes and now requires the additional Exit receipt, including runtime survival and playback cleanup.
- Convergence checked9 functional requirements,5 success criteria,8 story acceptance scenarios and the named architecture/constitution constraints. One HIGH partial gap (SC-003) was appended asT015: per-workflow mounted interactions. Implementation now adds actual keyboard Tab/Space and label-padding clicks to every renderer scenario, plus native focus/label checks. The complete rerun remains required before closingT015.

- T015 complete: clean sourcee45f90a653314ff61bd119f77f2f5143d2eab2f9 passes all96 scenarios with actual Tab/Space focus and label-padding activation, values restored, and original receipts/PNGs published unchanged. Capture/build/check took20.047seconds. A relative-label locator defect in the test harness was corrected before this successful run.

- Final local site check/test/build and repository check/test pass with the updated captures. UTF-8/no-BOM, LF and mojibake checks report no problems across the changed text files; git diff whitespace validation passes.

## PR/native validation

- PR153 opened onb0cdad3f6f8da184c2036ad956372201dd3a15e7. Automatic code review (round1) completed with zero findings and no review threads. Both issues and the PR are in Project7 review under milestone21.
- Initial source native jobs on Windows/Linux exposed a test-only relative companion path: Go tests run from cmd/insonic-desktop, while the fixture used build/native relative to that directory. The test now supplies an absolute repository-root media-tools path when no override is elected. A local regression with INSONIC_LIBRARY_TOOLS_FILE removed passes in4.407seconds. No gate was removed or relaxed.
- The initial extracted Linux desktop package job passed its actual GTK Exit, ordinary close, playback cleanup, exact shared runtime/work survival and acquisition completion assertions, as well as the compact-control native journey.

- Initial Windows archive qualification reached the new native Exit probe after passing the compact native journey, then rejected the application window because the probe filtered for the underlying winc_Form class. Inspection of pinned Wails2.16.0 window.go confirms its application class is wailsWindow. The probe now selects only its own PID, exact application title and actual native menu, retaining ambiguity rejection and mnemonic activation without depending on that internal class. The corrected Windows probe compiles, and the override-free fixture test passes again.
- Windows setup, maintenance and interactive installer lanes passed on the initial head. The archive lane failure above and the two source test-fixture failures are addressed together before rerunning full qualification.

- Round2 security review onf415a08241bc90878e92342ed5a85bb295f058d3 completed with zero findings and no unresolved threads; both authorized rounds are exhausted. Corrected Linux source and package native qualification pass.
- Corrected Windows source qualification passes the Go fixture and compact native journey, then reports that SC_KEYMENU did not enter menu mode. Pinned Wails calls OnDomReady before its initial ShowWindow; process.Capture correctly supplies a hidden STARTUPINFO, whose first-show override is documented by Microsoft (https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-showwindow). The explicit disposable Exit child now queues normal Wails WindowShow and waits for a visible enabled owned window before menu messages. It retains the native menu-mode/owner assertions and adds bounded numeric state diagnostics; there is no direct callback fallback and ordinary application startup is unchanged.
