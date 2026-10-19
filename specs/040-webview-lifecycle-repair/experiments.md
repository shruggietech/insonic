# Controlled experiments

## E001: actual native BOOL ABI (2026-10-19)

**Hypothesis**: The production wrapper supplies a pointer instead of native FALSE/TRUE.
**Control**: Unchanged maintained v1.0.22; same native callback-vtable test and Go toolchain. No browser/window is created.
**Prediction**: False/true both fail canonical argument equality before correction. Authoritative v1.0.23 supplies 0/1 and keeps HRESULT behavior unchanged.
**Bound**: One false and true invocation plus one E_FAIL invocation per selected production wrapper; no timing or GUI sampling.
**Result**: Predicted failures observed before correction; identical regression passes after correction. All397 v1.0.23 upstream hashes verify, exact two-file upstream delta, prior maintained lifecycle patch preserved.
**Conclusion**: Native startup ABI violation reproduced and corrected. This does not by itself establish the internal cause of all #154 failure signatures.
**Next decision**: Observe final native response/deferral completion and original end-to-end release timings in one bounded source experiment before declaring incident repair.

## E002: paired ABI and final response boundary (planned, not dispatched)

**Hypothesis**: A directly relevant upstream native BOOL correction changes the observed startup behavior. Any remaining pre-DOM failure can be separated into unfinished/failed native response completion versus successful completion followed by failed browser consumption. Release timing identifies the original budget remaining after root exit.

**Control**: Two freshly built GUIs from the exact clean S040 source share the same observation code, compiler, flags, frontend and verified companion payload. The control uses a Go build overlay containing exactly the historical upstream edge Controller3 file (SHA256 788ed4d41dc722f387c67e0ba873608ae5ec5debd338316ba1e81440021782c5). Its sole linked source difference restores the old pointer-to-bool expression. The source file is not edited, and the overlay is rejected unless its entire historical file hash matches. Both payloads retain the same executable basename and layout. Separate owned fresh profiles are required for every child.

**Corrected arm**: The checksum-verified v1.0.23 wrapper passes0/1, with all maintained lifecycle patches and the same observers. No concurrent CLI workload is elected. Both arms mirror startup-hide and use the same no-workspace close path.

**Predicted outcomes**: A failed old-ABI control with successful corrected observations supports the upstream lead but does not prove a shared private cause for every signature. A corrected pre-DOM failure after all native publication/Complete calls return successfully moves the investigation beyond host response completion. Missing/failed native operations identify the first host boundary. A corrected release failure retains actual owned members and its remaining budget. All passing samples establish only those executed cases.

**Bounds**: One disposable Windows source job. Each predeclared arm has at most150seconds/120children (300sampling seconds/240children total); each arm stops at its first failure. A known failed control is retained before the corrected arm proceeds as planned; this is a comparison, not a retry that clears failure. A corrected-arm failure fails the workflow. Original12-second watchdog and18-second combined root/release deadline remain unchanged. Unused control time is not transferred to the corrected arm. Do not automatically rerun or dispatch packaging for diagnosis.

**Provenance/privacy**: Control and corrected GUI hashes, exact branch source and historical overlay identity are retained; every comparison receipt remains feature_qualification=false. Control bytes cannot confer product/package acceptance. Fixed counters/status/timing and function symbols only; no request content, URLs, secrets, raw pointer identities or ambient-process targeting.

**Local validation**: Four new regressions verify full historical-file identity without editing production, explicit build overlay, separate fixed arm budgets/expected failed-control retention, and corrected failure remaining failed. All28 selected startup diagnostic tests pass. Native enabled/disabled fake-vtable tests validate observation neutrality.

**Result**: Not executed.
**Next decision**: Inspect both arms and first failed boundaries. Write a new falsifiable experiment before any further dispatch; never infer rare-failure absence from a finite passing count.
