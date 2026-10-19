# Research and decisions

## 2026-10-19

- Decision: Explicit registry guidance/example metadata. Rationale: commands.go derives examples from display operands and uses input.json for enum options. Alternatives: duplicated prose or a new parser framework.
- Decision: Execute emitted examples in the existing installed harness with a coverage ledger. Rationale: #119 requires semantic execution. Update root-help consumers and aliases together.
- Decision: Structured fixed validation causes and known constraint/location messages. Rationale: the schema validator retains details discarded by main.go. Never serialize raw validation errors, which can echo values or unknown property names.
- Decision: Preserve 1..600000 durations through schema/catalog forward migration. Rationale: parser/help promise ten minutes; request, catalog and DDL cap at one minute. Freeze historical schema-9 authority and retain old snapshot restoration.
- Decision: Unicode code points for public character limits; separate byte budgets unchanged. No normalization.
- Decision: Preserve exact model identities and quote/escape them in human output. Rationale: manifest validation is reused for retained installs/snapshots and identities hash original bytes. Global tightening can strand history.
- Decision: Suppress only unestablished failed results before zero work/publication converters manufacture maps. Never delete valid false/zero fields globally.
- Decision: Reserve summaries for identity, every task conclusion/correction, continuation and verbosity before optional details. Raising limits would leave ordering defective.
- Decision: Display valid UTF-8 interior and label incomplete byte boundaries in capture pages. Preserve exact runtime offsets/base64 and authority.

Three read-only Spec Kit research agents examined help/qualification, renderer/receipt and validator/catalog seams. Recommendations were compared with approved scope before adoption.

## 2026-10-19: native repair scope and demonstrated failures

Owner approval adds #154 repair. S038 already retained exact child streams and ran more than one hundred focused native launches without recovering the original HRESULT; its checkpoint explicitly deferred root repair. S039 now captures 800700aa (resource in use) during controller startup in Alt+F4, close and menu fixtures. A separate menu attempt reached its 12-second startup watchdog with no DOM-ready/native-action checkpoint or HRESULT. Its shutdown marker follows fallback Quit and does not prove a native action. These signatures remain distinct until evidence links them.

Microsoft documents that WebView creation can fail and recommends re-creating the environment when retrying (except invalid-state), while its user-data-folder guidance requires observing browser/child exit before deletion and notes that files may remain in use after host exit. These are investigation constraints, not proof that profile ownership is the cause here. Primary sources: https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2environment and https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/user-data-folder .

Inspect the existing Go Capture and Python ProcessTree job-close semantics, browser membership at root exit, and profile selection. Compare a controlled old boundary with the corrected owned lifecycle before adopting a fix. Preserve all real native qualification and proof gates.

### Demonstrated owned-job boundary

A real Windows helper regression failed in 0.15 seconds before implementation: the root returned zero, Capture closed its KILL_ON_JOB_CLOSE job, and the still-running owned child never wrote its natural-release marker. The new explicit Windows Capture wait mode observes job ActiveProcesses until zero under the original caller deadline; timeout remains failure and cleanup termination cannot establish release. The same regression then passed, along with a held-child deadline regression. This proves the defective ownership transition, not the private WebView2 implementation cause of HRESULT 800700aa. The normal desktop keeps its existing Wails profile defaults; native Exit fixtures use per-case owned profiles and reuse them only after observed release. Each of six Windows action/workspace cases runs cold and warm. All original native/playback/durable assertions and package byte/source gates remain.

The SDK accounting ABI and query contract are verified against [Microsoft job accounting](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_basic_accounting_information) and [QueryInformationJobObject](https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-queryinformationjobobject). Current-source diagnostic 39237018511 remained non-reproducing (48 Exit children, historical companions), and therefore does not prove the correction. Real corrected-package Windows qualification and full CI remain pending.

### Native menu activation convergence

The corrected-profile archive run observed six owned descendants exit naturally before warm reuse. Its next menu child reached DOM readiness and action injection but never completed Exit before the watchdog. Posted WM_CHAR delivery to the owner window is insufficient evidence that its modal menu selected Exit. Use actual Alt+F and X through SendInput, observe the active owned menu, and retain a real callback checkpoint plus normal-return assertions. This follows [Microsoft keyboard input routing](https://learn.microsoft.com/en-us/windows/win32/inputdev/about-keyboard-input) and [SendInput](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput); callback alone still cannot qualify success. The pre-DOM watchdog and controller-busy signatures remain separately recorded.
