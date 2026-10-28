# Research: Import interruption recovery

**Date**: 2026-10-27

## Current admission authority

**Decision**: Target `admission.go` canonical import and CommitAdmission/CommitElectedAdmission.
**Rationale**: Legacy `captureOrRecover` tests reconstruct a different path. Current admission captures anew and `freshAdmissionCandidate` retires candidates even when a complete report survived abrupt loss. Native failure tests must distinguish that from observed ordinary failures, which already retire candidates intentionally.
**Alternatives**: Reusing only the legacy helper or claiming terminal Ctrl+C recovery would not cover the current import/runtime boundary.

## Deterministic external boundary

**Decision**: Hold the final configured canonical ffprobe, arm an owned source-file open barrier, then release the probe and observe its break after durable report publication but before admission.
**Rationale**: The original source is reopened for its final digest check at `service.go identity`, after metadata publication. Ordinary tool configuration supports hash-pinned wrappers/interpreters; no product fault flag is needed. A hidden local Windows prototype proved an actual child open blocks and resumes on release.
**Alternatives**: SQLite polling/lock races and timed kills cannot prove the selected boundary. PostgreSQL triggers would qualify a different backend. An internal environment fault switch would expand shipped intervention behavior.

Windows uses a modern RW oplock with ACK_REQUIRED, overlapped handles and drained asynchronous lifetime; close only after the selected runtime is dead. [Request API](https://learn.microsoft.com/en-us/windows/win32/api/winioctl/ni-winioctl-fsctl_request_oplock), [break/output semantics](https://learn.microsoft.com/en-us/windows/win32/api/winioctl/ns-winioctl-request_oplock_output_buffer).
Linux uses a separately owned helper with main-thread SIGIO handling and F_SETLEASE/F_WRLCK on an owned regular file. Preserve the kernel lease-break-time and finish before its forced-break boundary. [Linux lease contract](https://man7.org/linux/man-pages/man2/F_SETLEASE.2const.html).

## Candidate reuse

**Decision**: Recover only a complete canonical/report pair, validate report bytes and source digest/size before reuse, verify canonical bytes, and reuse original captured metadata/facts rather than re-extracting them.
**Rationale**: The existing report binds original source identity and canonical provenance; add an optional exact canonical publication ID/digest/size binding before new report publication. This permits complete-pair verification without a new catalog schema/journal or duplicate admission model. Reports predating that binding retain existing recomputation. New attempts still commit under existing work and catalog fences.
**Alternatives**: Re-encoding may change container bytes and replaces capture time/provenance. Silently accepting changed input would change the interrupted operation's authority. Partial candidates without a complete report remain recomputable under existing behavior.

## Immutable release compatibility

**Decision**: Require the new recovery receipt for current/unknown packages, with only exact original v1.0.2 package-receipt hash matches allowed to lack that new field. Keep existing qualification checks and v1.0.1 historical profile unchanged.
**Rationale**: The published v1.0.2 manifest (SHA-256 `02a55b6ec51855881a1fc7592b002c5d3ce5d5dce0bf09985094da27beee3449`, revision `c3fe07295b2e74188ed0d074d14e831d730b0332`) predates this field. New tooling must not invent evidence for immutable binaries or repeat #185's compatibility regression.
**Alternatives**: An unconditional new requirement breaks original releases; a version-range or caller-controlled opt-out weakens unknown/current package enforcement.

## Prerequisites and delegation

Pinned source media preparation already writes `build/native/media-tools.json`; package tooling has matching extracted companions. Pass that maintained authority explicitly for source fixtures. Qualification uses owned profiles and ProcessTree lifetimes. Spec Kit plan directed one read-only research agent; it identified the modern/legacy distinction and platform barrier protocols. No unresolved research questions remain before task generation.
