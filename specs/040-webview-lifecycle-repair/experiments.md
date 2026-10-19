# Controlled experiments

## E001: actual native BOOL ABI (2026-10-19)

**Hypothesis**: The production wrapper supplies a pointer instead of native FALSE/TRUE.
**Control**: Unchanged maintained v1.0.22; same native callback-vtable test and Go toolchain. No browser/window is created.
**Prediction**: False/true both fail canonical argument equality before correction. Authoritative v1.0.23 supplies 0/1 and keeps HRESULT behavior unchanged.
**Bound**: One false and true invocation plus one E_FAIL invocation per selected production wrapper; no timing or GUI sampling.
**Result**: Predicted failures observed before correction; identical regression passes after correction. All397 v1.0.23 upstream hashes verify, exact two-file upstream delta, prior maintained lifecycle patch preserved.
**Conclusion**: Native startup ABI violation reproduced and corrected. This does not by itself establish the internal cause of all #154 failure signatures.
**Next decision**: Observe final native response/deferral completion and original end-to-end release timings in one bounded source experiment before declaring incident repair.

## E002: paired ABI and final response boundary (2026-10-19)

**Hypothesis**: A directly relevant upstream native BOOL correction changes the observed startup behavior. Any remaining pre-DOM failure can be separated into unfinished/failed native response completion versus successful completion followed by failed browser consumption. Release timing identifies the original budget remaining after root exit.

**Control**: Two freshly built GUIs from the exact clean S040 source share the same observation code, compiler, flags, frontend and verified companion payload. The control uses a Go build overlay containing exactly the historical upstream edge Controller3 file (SHA256 788ed4d41dc722f387c67e0ba873608ae5ec5debd338316ba1e81440021782c5). Its sole linked source difference restores the old pointer-to-bool expression. The source file is not edited, and the overlay is rejected unless its entire historical file hash matches. Both payloads retain the same executable basename and layout. Separate owned fresh profiles are required for every child.

**Corrected arm**: The checksum-verified v1.0.23 wrapper passes0/1, with all maintained lifecycle patches and the same observers. No concurrent CLI workload is elected. Both arms mirror startup-hide and use the same no-workspace close path.

**Predicted outcomes**: A failed old-ABI control with successful corrected observations supports the upstream lead but does not prove a shared private cause for every signature. A corrected pre-DOM failure after all native publication/Complete calls return successfully moves the investigation beyond host response completion. Missing/failed native operations identify the first host boundary. A corrected release failure retains actual owned members and its remaining budget. All passing samples establish only those executed cases.

**Bounds**: One disposable Windows source job. Each predeclared arm has at most150seconds/120children (300sampling seconds/240children total); each arm stops at its first failure. A known failed control is retained before the corrected arm proceeds as planned; this is a comparison, not a retry that clears failure. A corrected-arm failure fails the workflow. Original12-second watchdog and18-second combined root/release deadline remain unchanged. Unused control time is not transferred to the corrected arm. Do not automatically rerun or dispatch packaging for diagnosis.

**Provenance/privacy**: Control and corrected GUI hashes, exact branch source and historical overlay identity are retained; every comparison receipt remains feature_qualification=false. Control bytes cannot confer product/package acceptance. Fixed counters/status/timing and function symbols only; no request content, URLs, secrets, raw pointer identities or ambient-process targeting.

**Local validation**: Four new regressions verify full historical-file identity without editing production, explicit build overlay, separate fixed arm budgets/expected failed-control retention, and corrected failure remaining failed. All28 selected startup diagnostic tests pass. Native enabled/disabled fake-vtable tests validate observation neutrality.

**Dispatch**: One run [39329500568](https://github.com/shruggietech/insonic/actions/runs/39329500568), attempt1, job118323827024, exact source31f3959e41a1428eea9cc303b682c135bf10981d. Dispatched2026-10-19T21:41:49Z with startup_source/cold/hide/abi_control=true and startup_load=false.

**Result**: Completed successfully. Each arm executed120fresh children with zero observed failures. Artifact12223399860 (188114bytes), digest abeb2d1063ebd13b15b1a68108d249c4d9c3bf614b0665bd8d06cab9de10828f. Corrected GUI342223e66f3fe915e49c4c7668443a9bdccaee23cf372acd686b59ab910ff0e4; historical control65afb7b1f545a7f6bae0a68b85bc60f41cf0d6e3a032b8a95a5857cae9e8194b. All receipts retain feature_qualification=false. No failed boundary was captured. Both arms passing do not establish that the corrected BOOL caused #154, nor rule it out as a contributing defect. The historical arm executes first, so global-machine warmup/order is an additional confound: its first launch consumed14.875seconds before root completion, versus0.453-0.656seconds in the later corrected arm. Natural release still completed within the original remaining3.125seconds for that first control. No speed improvement is attributed to the ABI correction.
**Next decision**: Inspect both arms and first failed boundaries. Write a new falsifiable experiment before any further dispatch; never infer rare-failure absence from a finite passing count.

## E003: restore the historical concurrent CLI workload (planned)

**Evidence selecting this experiment**: The retained failed cold028 diagnostic explicitly records concurrent_cli_contract=true and includes the real documented CLI workflow/help runner (168executed examples,96.906seconds for examples). E002 explicitly excluded that workload and neither ABI arm failed. Load sensitivity is therefore a concrete untested variable, not an arbitrary rerun. No causal claim follows from the earlier single occurrence.

**Hypothesis/prediction**: Reintroducing that exact diagnostic workload on the corrected S040 source may reproduce pre-DOM failure. The new response counters will separate host publication/deferral completion from subsequent browser consumption. If a release failure occurs instead, preserve its original remaining budget and owned process membership. E002 is the idle reference on the same runtime source, not a simultaneous randomized control. The existing load helper submits one worker before sampling and joins it afterward; it has no start barrier or per-case overlap timestamps. concurrent_cli_contract=true establishes election, and its receipt establishes completion, but neither establishes that the workload was still running at each child or a failure. Do not label a recurrence load-caused or even load-overlapping without additional timing evidence.

**Control/source**: Reuse exact immutable runtime/tool head31f3959e41a1428eea9cc303b682c135bf10981d, startup_source/cold/hide/load=true, startup_abi_control=false. Same pinned compiler and verified incident companions; the source-built CLI exercises the existing documented contract only. Fresh owned profiles and close path match cold028. No loader, callback, deadline or SDK change is elected.

**Comparison limits**: Relative to an E002 corrected arm, the single load path also rebuilds the sibling CLI, initializes its separate workspace and restores the historical300s/240-child envelope. It is a historical-workload reproduction, not a randomized single-variable causal test. No direct GUI library/profile collision is demonstrated: the GUI bridge has no workspace, each WebView profile is fresh, runtime IPC keys include workspace identity, and playback uses its own ephemeral loopback port. Shared machine CPU/memory/storage and native companions remain possible contention channels.

**Bounds**: One disposable source job, at most300sampling seconds/240fresh children, stop at the first failure; the existing concurrent contract runner executes once. Preserve12-second watchdog and18-second combined deadline, original failure and feature_qualification=false. Do not rerun this workload automatically after either outcome.

**Next decision**: A reproduced first-response failure identifies its host operation; complete successful publication followed by pre-DOM stall shifts work to actual browser/runtime behavior. If all observations pass, record that this workload did not reproduce the incident, stop repeating it and use independently demonstrated native contract violations plus the ordinary condition matrix for the next repair decision. No closure of #154 based solely on a finite passing sample.

**Dispatch**: One run [39330500115](https://github.com/shruggietech/insonic/actions/runs/39330500115), attempt1, exact source31f3959e41a1428eea9cc303b682c135bf10981d, created2026-10-19T21:49:40Z. Prediction and limits were published before dispatch in [issue comment6278156090](https://github.com/shruggietech/insonic/issues/154#issuecomment-6278156090).

**Result**: Failed as designed at first recurrence, child029 after29passing children, original12-second watchdog, host exit1 at13.156seconds; no output truncation/incomplete capture. Artifact12224592039,28924bytes, digest1163d7e7ad9ffc68146863385efbe479a74e965f559dd3a40a0f2c37a28940fe. The corrected GUI hash342223e66f3fe915e49c4c7668443a9bdccaee23cf372acd686b59ab910ff0e4 exactly matches E002; rebuilt CLI2cd1238bd8c48d88834333913ef15bbad252c1dc9b822d5397bafb2cd2df04b9. The elected workload later completed its168help examples (70.094seconds), but no exact overlap receipt exists.

**First boundary**: Navigation requested/accepted1/1, resource callbacks6/6; middleware document1/1, script1/1, other2/2; navigation completion0. For the first native resource only, assignment/headers/status/reason/content/deferral/completion all entered/returned1/1, HRESULT0, off-thread0; status200,489content bytes, one copied stream/no failure. UI is idle GetMessage, no response-worker stack among the3captured Wails stacks. This rules out a missing/failed first native response publication or Complete for this occurrence, and rules out the BOOL correction as a complete repair. Later native responses/browser consumption/event delivery remain unobserved; six returned resource callbacks do not prove six completed responses. Nonzero host exit means the retained7job members are failure-cleanup evidence, not a separately observed natural-release timeout. No rerun.

**Decision**: Continue the independently reproduced native callback COM correction, and audit the exact post-response/script/navigation event boundary before selecting further observations. Do not repeat the unchanged concurrent-workload test.

## E004: corrected callback identity and native lifetime (planned)

**Evidence/hypothesis**: E003 reproduces startup failure after successful first response completion. Actual9callback native-vtable/GC regressions independently demonstrate invalid interface-query and retention behavior, including the navigation callback. A focused existing-combridge correction should preserve normal callback dispatch, reject nonexistent interfaces, retain native-owned targets and cancel late delivery. Its effect on the recorded browser stall remains to be observed.

**Control**: The retained E003 corrected-BOOL GUI failure at31f3959. The new candidate changes the9active callback COM implementation and associated creator/shutdown ownership only; frontend, loader selection, response observations, concurrent-workload helper and native deadlines stay identical. Source/notice authority must verify and all new unshown native regressions must pass before dispatch. The actual GUI/source/head hashes are recorded at dispatch.

**Prediction/outcomes**: A recurring failure remains failed and disproves this as the complete repair; inspect whether the first response and navigation/script boundaries change. A complete passing sample supports only executed stability with the concrete native contract fixes; it does not reveal the private internal cause or prove rare-failure absence. Later native response/browser/event boundaries remain explicitly unverified when no failure is captured.

**Bounds**: One source job using source/cold/hide/load=true, ABIcontrol=false; maximum300sampling seconds/240fresh close children, stop first failure, original12-second watchdog/18-second combined deadline and one existing concurrent contract invocation. Same exposure/confound limits as E003. No automatic rerun.

**Next decision**: If failure recurs, observe the5unobserved initial response completions and runtime-ready boundary before another hypothesis change. If this bound passes, run the unchanged actual cold/warm/library/native-action matrix and ordinary final package/CI qualification. Do not close #154 merely for passing repetitions; its post-merge acceptance remains separate.

**Result**: Not dispatched.
