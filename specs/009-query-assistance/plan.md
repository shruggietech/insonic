# Implementation Plan: S009 Optional query assistance

**Branch**: `codex/009-query-assistance` | **Date**: 2026-10-08 | **Spec**: [spec](spec.md)

## Summary

Deliver #13 with revisioned optional query assistance, complete inspectable proposals and explicitly elected read-only execution using shared direct query semantics. Local and hosted services use a versioned elected HTTP contract. Keep inference out of required CI through deterministic injected fixture adapters.

## Technical Context

Go 1.27.1, existing SQLite/PostgreSQL catalog schema 6, LadybugDB 0.21.2/ArcadeDB 26.9.1, Wails 2.14.0 and current React/TypeScript desktop. Native Windows/macOS/Linux packages. Existing setting_revision storage and receipt/snapshot validation; no new database migration or package dependency. Tests: repository Node/schema, Go acceptance/vet/affected race, frontend rendered/typecheck, Python integrity/package, site/offline export, real graph fixture and mounted source/package qualification. Every CI job retains ten-minute limit.

Bounds: prompt 16 KiB; provider output 256 KiB default and at most 512 KiB; provider deadline 30s default and at most 60s; query deadline 20s maximum; result at most 500 rows/512 KiB and default 100; explicit context at most 25 rows/32 KiB. IPC budget 90s covers configured phases and overhead. No whole-library content sent by default.

## Constitution Check

1. Approved scope: #13 only, no training/distribution or product eligibility controls. Pass.
2. CLI-first/shared runtime: query assistance-show/set/assist operations and desktop wrapper. Pass.
3. Current evidence: ephemeral suggestions/results; excerpt context is an elected current query, bounded before transmission. Pass.
4. Full supported adapters/credentials: named catalog settings are portable; graph validation honors selected dialect; opaque credentials with no fallback. Pass.
5. Truthful efficient delivery: deterministic fixtures, real existing graph checks, bounded two-round reviews and final exact-head CI. Pass.

Post-design check passes all five principles. Requirements checklist CHK001-CHK005 satisfied by spec, contracts and explicit bounds; no unresolved clarity gap. No extension hooks are installed.

## Decisions

- D01 Store nonsecret assistance settings through existing setting_revision with dedicated per-setting CAS/read helpers. Reusing file-based desktop overrides would omit catalog backup portability; a new schema table adds no benefit.
- D02 Use versioned `insonic-http` query-assistance POST for both local loopback and elected hosted routes. Arbitrary provider APIs require a protocol adapter; no implicit compatible provider.
- D03 Capture schema/capability/profile context briefly, release graph lock during provider work, fence settings/profile before accepting/executing. A provider never chooses routes or modes.
- D04 Reuse one context-aware query runner for direct and assisted queries. Native generated output requires structural read validation and reachable EXPLAIN before auto-run; unreachable native proposals are inspectable with pending status. Normalized queries retain current catalog fallback.
- D05 Enforce proposal bounds without silently rewriting meaning: normalized pagination exceeding configured limits is rejected; native results remain constrained by established backend caps, with smaller returned assistance caps diagnosed. Existing current hydration/time strings are preserved.
- D06 Request context links runtime owner, transport disconnect and deadline. Inference timeout and query timeout use that context; context-aware phases check cancellation before/after bounded in-memory hydration.
- D07 Provider input includes actual Entity/EvidenceLink storage schema, normalized vocabulary and selected capabilities. Explicit context is collected via a validated current direct query, sliced to elected bounds with truncation disclosed; no cached transcript copy.
- D08 Extract a full editor proposal loader; retain hidden filters/order/typed params and reset saved-query identity when adopting. Prompt/config/editor changes invalidate pending assistance synchronously.
- D09 Production provider execution refuses CI; fixtures inject a deterministic adapter at runtime construction. Explicit native qualification uses only that fixture, not an ambient provider or model.

## Project Structure

`internal/assistance/` protocol/config/HTTP; `internal/catalog/assistance_settings.go` existing-table CAS; `internal/app/assistance.go` shared dispatch and query integration; `internal/runtime/` contextual dispatch/deadlines; `internal/contracts/explore.go` operations; `cmd/insonic/explore.go`; `desktop/frontend/src/assistance.tsx` and `explore.tsx`; versioned schema/docs; existing native qualification journey.

## Execution

Specify/clarify/checklist -> research/design -> tasks -> blocking analyze -> tests then implementation -> acceptance/converge -> commit/push/official PR -> resolve initial reviews -> one final requested round -> exact-head green CI -> owner squash merge handoff. Implementation stays sequential; read-only research agents examined runtime/catalog and CLI/UI integration.
