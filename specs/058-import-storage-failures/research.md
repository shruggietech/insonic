# Research: Local import storage-failure recovery

**Date**: 2026-10-28

## Native fault boundaries

**Decision**: Use the existing copyStableBound writer and Filesystem.persist seam, plus a private stage finalization helper.
**Rationale**: Real files can receive partial writes while a test wrapper returns ENOSPC/short-write/flush/rewind errors. Existing code ignores final rewind and cleanup errors; redaction converts capacity errors to generic unavailable despite an existing insufficient_space response.
**Alternatives**: Shipped environment fault flags, a new journal and a new storage adapter are unnecessary.

## Actual filesystem refusal

**Decision**: A configured hash-pinned final probe hashes canonical output and creates an owned regular file at its exact new digest parent, then real publication fails. Remove only the recorded inode/hash-bound blocker and submit another ordinary import.
**Rationale**: This uses ordinary tool configuration, actual OS directory creation refusal and tiny deterministic fixtures on Windows/Linux without privileges or filling a volume. The accepted sentinel has a different digest.
**Alternatives**: Native quota/mount/ACL fixtures require privileges or heavier platform state. Timed polling cannot prove the desired boundary. Report publication is covered natively because its capture-time digest is unavailable at the probe gate.

## Public retry semantics

**Decision**: Preserve partial batch behavior and use a distinct new import request after failure clears.
**Rationale**: The service returns a partial result, but application work finalization marks failed items as failed work. An actual initial run and internal/app/work.go confirm this distinction. The campaign records that result and verifies a distinct ordinary re-import; repeated native Execute alone establishes no public retry behavior.
**Alternatives**: Changing batch/work semantics would expand scope and risk existing contracts.

## Evidence compatibility

**Decision**: Add a strict nested storage receipt for current/unknown candidates while retaining exact immutable original v1.0.2 receipt hashes and the existing v1.0.1 historical profile.
**Rationale**: Historical bytes cannot acquire new runtime evidence. No version-wide bypass is acceptable.

## Windows path authority

**Decision**: Anchor the derived object relative path to the resolved owned library root before exact inode/hash-bound removal.

**Rationale**: Initial Windows CI rejected equivalent resolved/absolute path spellings. The positive regression and a real short-path alias campaign pass with a shared resolved anchor, while an outside-root resolved target remains refused. This changes fixture cleanup authority only.

## Research authority

One Spec Kit-directed agent inspected these boundaries read-only. Parent inspection confirms the relevant copy, persistence, staging and admission behavior. No unresolved research question remains.
