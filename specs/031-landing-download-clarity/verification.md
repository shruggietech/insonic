# S031 verification

## 2026-10-15: specification and local verification

Spec Kit specify, clarify, checklist, plan, tasks, analyze and implementation prerequisites completed for `specs/031-landing-download-clarity` on `codex/031-landing-download-clarity`, scope #101, milestone 14 and Project slice S031. No extension hooks are installed. The analysis gate has no blocking finding.

Rendered regression checks first failed for the original ambiguous package labels and five competing primary actions. The implementation passes all four new rendered-state cases. Geometry/activation regressions cover undersized/overlapping/touching targets, clipped/obscured labels and exact native-link interception.

Local parity through `scripts/run-hidden.ps1` (CREATE_NO_WINDOW, redirected noninteractive I/O):

- Repository/brand integrity: 1251 text files checked, zero problems.
- Schema check: 25 schemas, 22 contracts and 30 examples, version 1.0.1.
- Root Node tests: 106 passed, zero skipped/failed. Restricted-run symlink EPERM was resolved by the authorized hidden launcher outside that sandbox; isolation tests were retained.
- Site Node tests: 14 passed, zero skipped/failed.
- Full site build: public and offline exports each passed the 111-page relative-link/anchor inventory. No new dependency or kit-byte update.

The five package destinations match the fresh official v1.0.1 API and the original release manifest whose SHA-256 matches that API. See [link inventory](link-inventory.md). No binary download/install or package publication was performed.

## Hosted-browser and delivery acceptance

Pending the authorized PR's Linux-hosted CI: full-page snapshot/released-fixture captures, eight viewport/theme combinations per state, actual 200% text, 320px 400%-equivalent reflow, target/label geometry, all-link inventory, actual Tab focus and intercepted trusted Enter/mouse/touch activation. Existing 180-second browser deadline, external HTTP/WebSocket blocks, owned cleanup and 660-second CI target remain in force.

No local Windows browser, personal profile or installer was launched. Browser screenshots/report will distinguish same-source released fixtures from real exported state. Neither the automated checks nor image inspection certifies assistive technology or other browser engines. Live deployed inspection follows an authorized deployment after merge; this PR handoff cannot establish deployed behavior.

Review cap: two rounds maximum for this PR. Final exact-head CI/review evidence will be recorded on the PR after the last code commit.
