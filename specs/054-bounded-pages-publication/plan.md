# Implementation Plan: Bounded documentation publication

**Branch**: `codex/054-bounded-pages-publication` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

## Summary

Replace the single content-heavy tree request with bounded incremental unpublished Git trees, uploading individual oversized or binary files as verified blobs. Verify the complete final tree before any publication commit/ref update. Add bounded redacted API diagnostics and carry one existing deadline through all staging and build requests.

## Technical Context

**Language/Version**: Python 3.12+ standard library.

**Primary Dependencies**: Existing release GitHub REST client and qualification timing module; no new packages or child tools.

**Storage**: Original accepted static archive and unpublished Git objects.

**Testing**: Python unittest Pages/release suites, real local Git object oracle, root Node checks/tests, affected site checks and hosted elected CI.

**Target Platform**: Maintainer Windows/Linux release tooling.

**Project Type**: Existing release CLI helpers.

**Performance Goals**: Complete 1,110-file/54 MB isolated export within maintained execution allowances; bounded request assembly without quadratic batch serialization.

**Constraints**: Existing archive 16 MiB per file/64 MiB total/10,000 files; tree JSON at most 1 MiB, individual base64 blob JSON at most 24 MiB. No retry clock reset, new live destination, external dependency or Windows child process.

**Scale/Scope**: scripts/pages.py, scripts/release.py, tests/pages_test.py, tests/release_test.py, current maintainer documentation and CHANGELOG.md.

## Constitution Check

Pre-research and post-design: all five principles pass. Preserve owner scope/merge boundary, existing shared interfaces, immutable original content/provenance and exact release authority, token confidentiality and selected destination, bounded meaningful verification and separate phase accounting. No inference, GUI or release publication belongs to this slice.

## Research and Decisions (2026-10-27)

1. Use incremental Git trees via base_tree after an initial tree with no base. This removes obsolete nonhistorical files without copying them from the existing public tree. Keep the existing authenticated API interface rather than introducing a local clone and credential-bearing subprocess.
2. Budget actual json.dumps UTF-8 bytes with existing serialization, including escaping and envelope overhead. Inline only fitting UTF-8 entries; upload large/binary content separately and require its returned Git SHA to equal the original bytes. Deduplicate identical uploaded content.
3. Fetch the complete staged recursive tree and compare confined unique paths, type, mode, size and SHA against the complete original mapping. No truncated inventory or unexpected entry is accepted.
4. Bound API network waits with remaining promotion time before and after every call. The existing artifact-producer allowance remains unchanged and is never renewed per batch or request. Serial mutative requests are paced at least one second apart within that allowance, following GitHub guidance; the original export needs 82 content-staging requests and must not replace a payload failure with throttling.
5. Report structured redacted HTTP error context for JSON operations and asset uploads. Bound the response read and diagnostic output; discard arbitrary request/response echoes, query strings and sensitive fields; redact the exact active token and recognizable credentials. Retain useful GitHub message/error codes.
6. Keep unmodified candidate/tag/asset/source/domain/environment gates, non-forced ref update and uncertain-response reconciliation, exact build identity and live verification. Tests must never contact a live write endpoint.
7. User explicitly authorizes push, official PR and up to two Codex rounds, and reserves merging. Autopilot overrides the skill's default pre-push pause and optional reviewer-checklist pause. Reviewer-owned checklist markers remain untouched by implementation.

## Project Structure

Feature records: spec.md, plan.md, research.md, data-model.md, contracts/publication.md, quickstart.md, tasks.md and concise checklists. Existing source/test locations above remain authoritative; no new service or persistent product state.

## Delivery

Tests fail before repair, implementation satisfies isolated transport and confidentiality failures, root/affected checks pass, then commit with Co-authored-by: Codex <noreply@openai.com>. Publish PR linked to #189, milestone and Project S054. Retain compact CI audit outside run artifacts before any full rerun. Resolve every review comment and trigger no more than a second Codex round; stop before merge.
