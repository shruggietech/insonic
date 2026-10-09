# Recording authority checklist: S011

**Purpose**: Review replacement and roster requirements for completeness and consistency.
**Created**: 2026-10-09
**Feature**: [spec](../spec.md)

**Review Ownership**: Reviewer-owned requirements-quality artifact. Checked items approve requirement quality, not implementation. The autopilot kickoff authorizes the separate requirements review before implementation.

## Authority and recovery

- [x] CHK001 Are stable identity, target skip and independent replacement elections explicit? [Completeness, FR-002..005]
- [x] CHK002 Are applicability, known/unknown clocks and non-transfer of acoustic mappings unambiguous? [Clarity, FR-004..008]
- [x] CHK003 Are queued revision freezing, accepted replay and cancellation/atomic failure semantics complete? [Coverage, FR-006..009]
- [x] CHK004 Are shared/current assets, owner inputs and active leases distinguished in retirement requirements? [Consistency, FR-009]

## Intentional context and interfaces

- [x] CHK005 Are undeclared/empty rosters, idempotency, uniqueness and optimistic concurrency specified? [Clarity, FR-010..012]
- [x] CHK006 Are roster membership, cue attribution, identity mappings and training evidence explicitly separated? [Consistency, FR-013]
- [x] CHK007 Are initial atomic membership and replacement retain/clear semantics specified across interfaces? [Coverage, FR-014..017]
- [x] CHK008 Are portability, required model-free checks, deployment scope and issue completion boundaries measurable? [Measurability, FR-016..020, SC-001..005]

## Notes

Generation leaves items unchecked. A separate reviewer evaluates the written requirements; implementation reads this gate without changing markers.
