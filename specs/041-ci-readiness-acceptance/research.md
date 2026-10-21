# Research: CI readiness and acceptance

## Decision: actual readiness owns consumption

The checked-out package-windows job needs only scope and starts its bounded transfer wait before the source producer. Its successful cold producer can exceed 180 seconds. Keep the helper bound; gate consumers on verified terminal readiness. Retain exact cache and source/recipe/compiler validation. Source inspection and independent Spec Kit research review agree.

GitHub documents prerequisite success and failure/skip propagation in [job dependencies](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-jobs) and PR cache restrictions/eviction in [cache reference](https://docs.github.com/en/actions/reference/workflows-and-actions/dependency-caching). No sibling PR cache is assumed available on main.

## Decision: separately measured phases

The owner approved separate preparation and 660-second qualification, reporting both and total. All qualification starts after the readiness boundary. This avoids excluding checks that ran during preparation. A 15-second aggregate finalization allowance is explicit and requires later terminal observation, not a promise about provider latency.

## Decision: use existing producer trust

No cross-run source import. Reusable ready validates probe success plus either verified warm upload or successful cold ffmpeg upload. Exact current-run transfer restore and independent candidate validation continue. Package inventories record resolved versions; they do not strengthen the compiler identity into an unsupported fully pinned toolchain claim.

## Decision: aggregate is always evaluated

Require explicit success for elected work, skipped for deliberate non-election, and failure on unknown/missing results. Documentation-only and maintenance live-site elections remain explicit. Query the exact run attempt and complete bounded job inventory for timing; reject ambiguous identities. GitHub [job](https://docs.github.com/en/rest/actions/workflow-jobs) and [run](https://docs.github.com/en/rest/actions/workflow-runs) APIs are read-only.

## Cold Go dependency preparation (2026-10-20)

Decision: prepare external and standard Go libraries under the exact CLI, launcher and desktop contexts before readiness, with an empty isolated cache and a same-attempt verified transfer. Independent planning research confirmed Go list without -export discovers the graph without compiling product packages. Main-module packages and executable dependencies are forbidden build targets.

Rationale: first hosted assembly had a Go cache miss and spent 206 seconds building, pushing interactive completion beyond 660 seconds and archive into its job timeout. Existing optional cache reuse cannot guarantee the cold case. CGO flags differ between package variants, so both native paths must be represented. Go cache does not detect changed external C libraries, making native/compiler identity necessary.

Alternatives: a warmed retry provides comparison only; moving application builds into preparation violates the timing contract; increasing native or job deadlines weakens the intended acceptance. Sources: [Go build caching](https://pkg.go.dev/cmd/go#hdr-Build_and_test_caching), [Go package listing](https://pkg.go.dev/cmd/go#hdr-List_packages_or_modules).

## Second hosted timing convergence (2026-10-20)

Run 39497295880 at head a6e58fd41cf4a7559824abc62573c2ba0ea31467 (tested source 126ce06e9504b02738c516ef45e15908a3df67f0) passed every product check, all four Windows scopes, joined package proof and candidate verification. The elected aggregate rejected total qualification beyond 660 seconds. Assembly fell from 206 to 101 seconds, but its setup still restored an unused default Go cache for 26 seconds. The setup EXE uses a distinct CGO-disabled, trimpath compiler context not covered by the five prepared contexts. Archive passed at 21:17:37Z; join ended 21:18:21Z, candidate 21:19:21Z and aggregate 21:19:34Z.

Next controlled change: prepare the setup compiler dependency context (excluding product code), disable default Go cache restoration only where the verified package dependency cache replaces it, begin Windows consumer setup after global readiness while enforcing a separate foundation/docs gate bounded by the original readiness plus 660 seconds, and join independent module/source verification and npm installations in parallel. Product compilation starts only after gates; source transfer helper remains capped at 180 seconds. All native cases, complete idle allowances, source/compiler/digest checks and isolated setup source rebuilds remain unchanged. Analyze gate: no timing clock reset, product work before readiness, verifier isolation change or weakened acceptance.

## Bounded native allowance and observation (2026-10-21)

The failed archive child in run 39530233546 missed its two-second UI join before input. Its later focus and DOM-ready logs do not demonstrate a permanent hang. The available-runtime lookup was 131.0.2903.86; the created environment version was not recorded. The first comparison changes only the inner allowance to ten seconds, within unchanged enclosing limits. A ten-second policy may be adopted after full acceptance even if the small comparison is causally inconclusive; do not demand a framework rewrite or statistical permanence proof.

Microsoft documents [GetUserObjectInformationW](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getuserobjectinformationw) UOI_IO as the desktop input observation, and [OpenInputDesktop](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-openinputdesktop) requires closing acquired desktop handles. Window-station names and session IDs alone do not prove native input suitability. Borrowed process station/thread desktop handles are not closed. Query failures remain explicit unknown observations; functional foreground/input checks remain authoritative.

The created [environment BrowserVersionString](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2environment?view=webview2-1.0.4258.31) is distinct from loader availability. Reuse the bounded owned-string/freeing helper from the maintained binding while leaving navigation and CDP behavior unchanged. Wails remains v2.16.0, audited upstream commit 457bf210d5eb976ff3726c8629f7725575163307; confirm local installed source against its locked module.

A context deadline does not await in-flight completion, and unhook return need not mean an already executing callback ended. These are audit constraints, not evidence that the existing helper is defective. This phase observes posting, callback admission, return and joined completion without redesigning the hook protocol. Any behavior correction needs evidence and a regression for the changed contract.

## Bounded comparison and accepted policy (2026-10-21)

Corrected paired run 39556417815 attempt 1 reused executable SHA256 41de92d9a0fa8590a5ecf877610b03d5fc3f98b4a06af2ee8a0f7288d0bb8a65 from ordinary run 39554068896 attempt 1, source 8ab2bb05fecc40f2cc1b30a22e1fecef1be68310. Both twelve-case arms passed with natural release; maximal joins were 0.2485661s (two-second baseline) and 0.0746852s (ten-second comparison). This does not causally explain the earlier two-second failures. Adopt ten seconds as the explicitly authorized fixture policy, conditional on full final ordinary acceptance. Neither diagnostic arm overlapped the ordinary archive CLI workload, so final ordinary qualification must retain that concurrent work. The same VM image was win22 20261012.333.1, session 2, visible WinSta0, Default UI/input desktops receiving input, and owned visible/enabled foreground windows before all actual inputs. Each created environment getter returned actual runtime 131.0.2903.86 with zero HRESULT; no browser-folder override was selected.

The failed ordinary aggregate also exposed a provider-record distinction: /actions/runs/39554068896 reports workflow creation 06:13:06Z, while /attempts/1 reports attempt-record creation 06:13:07Z against attempt start 06:13:06Z. Initial queue must use the identity-verified workflow event record, retaining the latter timestamp separately. Do not clamp negative queue durations or move the readiness boundary. With correct authority, retained 48-job metadata yields 258s preparation, 552s terminal qualification, 810s combined and zero initial queue. Timing passes; product acceptance still fails. Prior cold/warm campaign source, cache, preparation and product paths are unchanged. Its stored workflow creation/attempt-start metadata can be recalculated with this correction; it cannot certify the new executable. Exact-final-head campaign closure remains tracked before #161 closure.
