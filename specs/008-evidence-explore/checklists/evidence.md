# Evidence and exploration requirements checklist

Created 2026-10-08. Feature: [spec](../spec.md). Audience: author and PR reviewer.

- [x] CHK001 Are ownership/clocks/overlap/attribution/proposition semantics defined (FR-001 through FR-004)?
- [x] CHK002 Are replay/fencing/correction/unknown outcomes specified without stale assignments (FR-005, FR-013)?
- [x] CHK003 Are parity/native compatibility/typed parameters/read-only boundaries distinguished (FR-006, FR-007)?
- [x] CHK004 Are whole-library uncertainty and recording clocks/playback separately defined (FR-009, FR-010)?
- [x] CHK005 Are versions/CAS/layouts/accessibility/truncation specified (FR-011, FR-012)?
- [x] CHK006 Are workspace isolation/real backends/zero required model execution explicit (FR-014, FR-015)?
- [x] CHK007 Are shared CLI/GUI contracts/version alignment/#21 reconciliation complete (FR-008, FR-016)?

New items are unchecked pending requirements review, not implementation acceptance.
Autopilot requirements review: CHK001-CHK007 satisfied by explicit spec/plan/contracts. No implementation completion is asserted.
