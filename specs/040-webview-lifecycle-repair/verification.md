# S040 verification

## 2026-10-19: source and ABI baseline

Housekeeping: verified PR #156 squash merge c9ed48d, identical tree to 8d2f1c7; fast-forwarded main, deleted verified merged local S038/S039 branches, verified 11 closed issue/project Done states, closed milestone23 with zero open items. #154 remains open, assigned to milestone24/S040 In progress.

The native production-wrapper regression failed against v1.0.22 for both false and true: one native call each returned a noncanonical BOOL, with no wrapper error. The HRESULT propagation regression passed. Adopting authoritative v1.0.23 makes both exact ABI cases and the same HRESULT test pass. No browser or application window was created.

Go module authority: v1.0.23 tag e08e9397f04ad0b1050d7e5bdc8e27faddc87987, upstream sum h1:jmv8qhz1lHibCc79bMM/a/FqOnnzOGEisLav+a0b9P0=. All397 file hashes reconstruct that exact sum. The only upstream differences from v1.0.22 are the two Controller3 Boolean wrappers. The original maintained three-file modifications/four additions are preserved before further observation work.

The initially attempted import of unused pkg/webview2 fails at package initialization because its pre-existing callback signature exceeds uintptr. Wails imports pkg/edge and does not execute that generated API. The executable regression therefore tests the actual production edge wrapper; the generated counterpart adopts the checksum-verified upstream source but has no native execution claim. No unrelated generated API rewrite is elected.

An initial local lifetime-test command used the nested module filesystem path, which Go correctly rejected; the selected test is rerun through its actual module import path. This is a command-selection failure, not a product test failure.

Native incident causality and S040 final qualification remain unproved. No new hosted experiment has yet run.

The selected existing unshown native lifetime suite passes through its actual module import path, including Close/reference/apartment, menu focus, claimed accelerator, bootstrap observation and deferral release cases.

## 2026-10-19: original release-budget timing

T013 native helper regressions failed without timing evidence and passed after the additive report. One fixed2.5-second budget covers root plus descendants. Measured failed fast/slow roots left2.4241058s/1.6212023s and waited those remaining budgets; both retained live owned children and stayed failed. Natural counterparts, no-deadline compatibility and prior natural/cleanup/held-handle regressions pass (focused selection9.771s). Timing observes exec.Cmd.Wait completion including pipe joins, not precise kernel exit; it does not change deadlines or prove a WebView cause. Python targeted-capture equivalents preserve the original18-second budget for5s/17s synthetic root durations. All24 selected startup diagnostic tests pass.

The upstream ABI native regression also passes three race-detection repetitions.

## 2026-10-19: existing post-merge pipeline baseline

Read-only audit of already completed main CI39325568611 on c9ed48d:43jobs,20success,8failure,15skip. Windows assembly failed its same-run source-artifact deadline before native qualification; all four Windows lanes failed waiting for assembly. Media AV1 artifact upload and adapter fixtures also failed. This is not a post-merge native startup reproduction and does not establish a common browser cause. No retry was dispatched. The targeted source experiment remains independent of these package handoff failures.

An intermediate npm test invocation was made before SDK observation edits and source-manifest refresh finished; its stale-source rejection is expected and will be rerun after source settles. Copied module-cache read-only attributes on the two adopted source files were normalized to writable source. Temporary symlink permissions also require the authorized unsandboxed test path; no integrity tests are weakened.

## 2026-10-19: final native publication observation

T009/T010 add exact first-native-resource association and fixed entered/returned/HRESULT/thread counters for assignment, headers, status/reason, content publication and deferral completion. No WebView2 COM call is added/reordered and no address/body/URL is emitted. Borrowed identities retire after their acquisition/Release; owner state disappears on native destruction. Actual-environment version is not queried; the legacy runtime_version field remains explicitly an availability lookup.

Three actual native-vtable tests verify enabled/disabled behavior, operation ordering/count, all seven failed HRESULT boundaries, foreign-thread observation without rerouting, borrowed-identity retirement, reentrant snapshot access and real SHCreateMemStream ownership after buffer mutation and GC. Selected normal tests plus prior unshown ownership suite pass0.042s; the same selection with race count3 passes1.091s. No browser/controller/window is created by the three new tests. Maintained upstream scope is now exactly five modified originals and the same four additions, with all397 upstream hashes preserved against v1.0.23 and patched sum h1:JaxxLqAJMJO/4zi5VgV41qyJquTZeLfJONk87N9lDas=.

Current local checks: root check and145Node tests pass; full394Python tests pass before four additional paired-control cases, and all28 targeted startup-diagnostic tests pass afterward. Production-tagged desktop, host, process and ABI packages pass, with vet clean. Site17tests and build pass; a subsequent export check validates111pages. Initial sandbox-only workspace/directory creation errors disappear on the authorized unsandboxed path; no product logic changed for them.

The observer reads native thread identity; it adds no WebView2 COM calls. Both paired arms use the same observers.
