# S029 research

2026-10-14: Spec Kit plan dispatched independent read-only release and documentation research agents. Both found the same concrete gaps; root consolidated their findings before design.

- Decision: Finalize existing unpublished 1.0.1 manually in reviewed source. Rationale: prepare() intentionally rejects same-version/overwriting directories (scripts/release.py:168–177); version is already synchronized. Alternative: weaken prepare or invent 1.0.2, rejected as unnecessary and inconsistent with #97.
- Decision: Fold Unreleased changes and chronological decisions into the existing unique 1.0.1 section, update its date to the UTC cut date 2026-10-15, refresh only current references. Rationale: current frozen changelog predates S019–S028; historical files remain immutable.
- Decision: Mark verified-published 1.0.0 released in mutable manifest and apply released-entry index prose as a render overlay for every released entry in both targets. Rationale: current docs.mjs only overlays latest and release-status.mjs gates on release mode, so metadata correction alone leaves false unpublished prose. Alternative: rewrite historical Markdown, rejected.
- Decision: Keep current source 1.0.1 candidate; elect release-target build only for publication snapshots. Rationale: source readiness is not publication.
- Decision: Add existing site tests and diagram/UI qualifiers before release docs archival. Rationale: release.yml lacks checks present in PR docs CI; no new browser framework or network permission is needed.
- Decision: Preserve complete release.py trust/tag/draft/hash/Pages contracts and historical recovery. Select fresh squash-main source operationally rather than prohibit legitimate historical recovery globally.
- Decision: Reuse valid application/MSI screenshot receipts. Research verified captured input hashes; root package.json is a capture input and needs no edit for this release.
- Decision: Existing Pages/domain and explicit unsigned disposition persist. No owner-host installer or native UI operations.
- Tooling note: checklist's prerequisite helper requires plan.md although autopilot orders checklist before plan. Initialize the installed blank plan template to satisfy the helper, then complete checklist before substantive planning. No hooks/extensions are installed.

All research unknowns resolved. Source evidence: scripts/release.py prepare/collect/publish/published; .github/workflows/release.yml and ci.yml; site/lib/release-status.mjs and docs.mjs; docs/versions.json; current/root changelog and references. Official GitHub release metadata confirms v1.0.0 and remote tag lookup confirms no v1.0.1 at kickoff.
