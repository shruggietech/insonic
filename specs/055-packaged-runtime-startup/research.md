# Research: S055

## 2026-10-27

- Observation: main c5cfcb75 failed the JSON terminal running barrier with startup_launch. Original native stage/errno is unavailable; retained audit is linked from #191.
- Source defect: runtime.ensureInput treats a failed 500 ms live-owner probe as permission to launch. Its owner-lock check runs only after a child returns runtime_owner. The terminal job forbids breakaway while detached launch requests it. A slow probe can therefore unnecessarily enter a launch path unavailable to that owned client.
- Decision: demonstrate failed-first-probe/held-lock behavior using real IPC, then check authority before launching. Keep the aggregate ten-second clock and cold-start conflict reconciliation.
- Alternatives rejected: enlarging clocks hides arbitration; enabling breakaway weakens terminal descendant ownership; unconditional retries of arbitrary startup errors can hide genuine package failures.
- Decision: elect the existing CI-only sanitized startup hook for terminal children and retain correlated bounded failure evidence before cleanup. Missing native evidence stays unknown.
- Alternatives rejected: retaining complete temp trees or raw logs risks private values and creates junk; preserving passing receipts cannot explain failed operation.
- Verification boundary: a deterministic repaired race and green corrected-source package acceptance do not retroactively establish the historical Windows errno or owner state.

## 2026-10-27: First CI correction

- Exact-head attempt 40586738671 reached the Windows desktop timeout case and returned generic cancellation/exit 1 instead of wait_timeout/exit 4. Captured owner was running, and prior human/JSON/quiet clients exited successfully. This is a distinct failure from the historical startup refusal.
- Controlled virtual-time regressions reproduce transport expiry before context Err/Done notification. Decision: classify expiry from the existing absolute deadline while retaining explicit caller cancellation and concrete startup reports. No timeout value or acceptance assertion changes.
- The rejected terminal audit (220 s preparation + 604 s qualification = 824 s total), eight original ZIP identities, five complete terminal receipts and correlated failure are retained on PR #192 comment 6454336247 and recovered byte-for-byte before another run.

## 2026-10-27: Setup inventory correction

- CI 40589369443 setup scope failed before product execution in wait-inputs with Incomplete job inventory. Its exact page counts were not retained by the earlier helper. Terminal/archive qualification continued independently.
- Source enumeration rejected short pages or changing totals immediately while the job graph was still active. New regressions exercise both cases, original deadline expiry, changed readiness/identity, persistent incompleteness and duplicate IDs. All four failed before the repair.
- Decision: keep strict attemptMetadata rejection and use at most three additional complete resnapshots only in waitForQualificationInputs. Partial validated observations may retain the original readiness bound and reject failures, never accept work. Terminal audit remains strict. No policy time or required check changes.
