# S066 research decisions

Three read-only research agents examined publication authority, release qualification/retention and independent public verification as required by the installed planning skill. No research agent published, edited source or dispatched workflows.

## Source and trusted tooling

**Decision**: Reuse the primary checkout, committing S066 planning before switching cleanly to main for trusted operations.

**Rationale**: scripts/release.py trusted_history requires clean tooling ROOT at current protected remote main; source-root selects candidate data, not privileged verifier code. Current main and selected product source both equal the accepted squash SHA. Exact-main CI passed and local/remote v1.1.0 tags and release were absent at preflight.

**Alternatives considered**: Publishing from the S066 branch would violate the trusted-main contract. A second worktree is unnecessary while both authorities coincide; use isolated checkouts only if main advances.

## Candidate and timing authority

**Decision**: Prepare once through release.yml with publication/promotion disabled, retain its complete release-candidate-ATTEMPT artifact and explicitly validate both documentation archives as release-target.

**Rationale**: The release workflow elects release rendering. load_candidate validates all ten assets but alone permits candidate-target docs; publication and explicit verify_documentation(require_release=True) reject that substitution. Required jobs include docs, verified media readiness, engine/desktop runtime lanes, all platform package variants, Windows scopes/finalization and final collection. Publication/promotion jobs intentionally skip during preparation.

**Alternatives considered**: Pre-merge or ordinary CI archives have different source/render authority. Rebuilding at publication would lose exact accepted bytes.

Release has no CI shared-readiness or elected-acceptance aggregate. Preserve original job/step and receipt intervals, supported preparation/qualification timing, combined wall time and terminal metadata. Do not apply ci-acceptance --audit to a Release run or borrow another run's readiness timestamp. Workflow head and selected input revision are separate identities even when initially equal.

## Publication and documentation

**Decision**: Create the exact immutable tag after acceptance, publish original assets using concise S065 highlights, then promote original release documentation through existing Pages.

**Rationale**: publish requires matching local/remote tag, clean accepted source, trusted main and green applicable checks. Draft upload/readback precedes publication; retries reconcile identical state only. Pages preflight verified gh-pages:/, insonic.io, HTTPS/environment/static markers and prior head f2316b6bec2f53a6d67afe6db92bf853999a22f0.

**Alternatives considered**: Automatic publish=true rebuilding orchestration after an accepted preparation is unnecessary. Generic hosting or domain/environment changes are outside the existing route.

## Independent verification

**Decision**: Pair published into a new directory with explicit ten-asset comparison and verify-public against the exact promoted Pages commit. Add maintained bounded live rendering from exact Pages data.

**Rationale**: verify-public proves local candidate authority, public API asset identities and actual live documentation bytes, not a repeat binary download. published downloads every official asset and validates it; comparison to the original candidate closes the exact-byte claim. website verify-deployed supplies complete package/source fields for eight headless landing cases without rebuilding the site.

**Alternatives considered**: API metadata alone cannot support a binary-download claim. A raw release public receipt supports rendering but omits optional package assertion fields, so use the maintained website receipt. Owner desktop GUI and real acoustic/device checks remain unelected.

Installed Playwright 1.64.1 selects Chromium 1248. The matching full chrome.exe is GUI subsystem and supports the existing headless=true launcher; console headless-shell would risk a window because upstream Playwright lacks windowsHide. Use hidden parent I/O and the explicit full binary.

## Existing documentation discrepancy

**Decision**: Record a separate documentation follow-up for stale aggregate timeout prose and retain actual measured budgets in S066 evidence.

**Rationale**: Current policy provides 600 seconds for live verification and 900 for browser/cleanup, while releases.md mentions older 120/180 aggregates. These are qualification allowances, not ordinary product deadlines. No acceptance bypass or publication failure was demonstrated. Changing frozen accepted source during publication is unnecessary.

## Evidence-only validation and review

**Decision**: Run repository/schema/root tests and existing focused release/pages/public-verification tests; rely on fresh Release lanes for shipped native/package bytes and final applicable CI for S066 records.

**Rationale**: No implementation is changed or new behavior test is warranted. Keep all maintained security/integrity assertions. Explicit owner authorization overrides the normal pre-push/publish pause. Two external review rounds maximum and owner-only merge remain binding.
