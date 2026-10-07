# Requirements integrity checklist: S005

**Purpose**: Reviewer-owned requirement-quality questions; unchecked does not denote unbuilt code.
**Feature**: [spec.md](../spec.md)

- [x] CHK001 Are all persisted assignment locations and reference-only consumers unambiguous (FR-002/003/013)?
- [x] CHK002 Are cue intersections, sub-ms collapse, unknown duration and no-cue outcomes explicit (FR-004/006)?
- [x] CHK003 Are cancellation/CAS/retry/current replacement and physical cleanup distinguishable (FR-007/008/009)?
- [x] CHK004 Are maintainer-only inference and permitted deterministic CI paths separately defined (FR-005/012)?
- [x] CHK005 Are asset licenses, exact source/rendition/crop and independent fixture expectations specified (FR-011)?
- [x] CHK006 Are strict export refusal and bounded upstream loss-accounting failures stated (FR-010)?
- [x] CHK007 Are legacy schema/retention conflicts reconciled without a second assignment store (FR-013)?
\nReviewer: Cueson research agent; parent adopted explicit inward projection clarification on 2026-10-07. Checked items certify requirement quality, not implemented behavior.\n