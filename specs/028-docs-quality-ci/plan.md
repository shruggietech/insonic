# Implementation Plan: Reliable CI and usable documentation

**Branch**: `codex/028-docs-quality-ci` | **Date**:2026-10-14 | **Spec**: [spec.md](spec.md)

**Input**: S028/#93–#96. Milestone12. Release/publication #97 remains after merge.

## Summary

Restore deterministic real-backend lease fencing qualification, align active CI policy/limits to the owner-approved660seconds, provide manifest-driven static latest documentation aliases with full no-JavaScript content and preserve explicit archives, and audit/improve shared documentation presentation across mobile/desktop/public/offline. Link the README company attribution. Preserve1.0.1 candidate and governed kit identity.

## Technical Context

**Language/Version**: Go1.27.1, Node>=22.12, Python3, Next.js16.4.0/React19.3.0, committed Spec Kit1.0.8 Python integration.

**Primary Dependencies**: Existing MarkdownIt/highlight.js/Mermaid pipeline, local governed brand tokens/fonts, playwright-core1.62.1 with installed headless Chromium.

**Storage**: Static manifest/HTML/CSS/local assets; existing real SQLite/PostgreSQL catalog fixtures. No new persistent product state.

**Testing**: npm run check, npm test, site tests/build/check; real catalog tests and vet; portable Python models; isolated browser accessibility/keyboard/reflow/theme/file checks; final hosted Windows/Linux product/package/candidate lanes.

**Target Platform**: Existing static Pages site and file-based offline help, current/historical documentation. Native Windows tests only on disposable CI.

**Project Type**: CLI/runtime plus static documentation site.

**Performance Goals**: Final exact-head CI<=660seconds; bounded headless validation with all integrity gates retained.

**Constraints**: No personal browser profile/window control; hidden noninteractive local child launches; UTF-8/noBOM/LF; preserved historical source/schema bytes; no release/merge/deployment.

**Scale/Scope**: Four issues, all documentation templates and representative content types, all exported pages overflow/structure scan and320/360/390/768/1440 viewport evidence.

## Constitution Check

Owner authority explicitly resolves the old ten-minute rule: amend active policy to eleven minutes and version constitution3.0.0->3.0.1. Native/product operations, backend security/tenancy, schema/software1.0.1, historical evidence and package/source identity remain unchanged. Static alias rendering preserves portability and avoids server-only redirects. Brand governed bytes remain untouched; semantic values/recipes supply visual changes. No new human verification gate or product restriction is introduced. All four journeys remain required. No unresolved CRITICAL conflict remains.

## Project Structure

Planning: spec.md, research.md, data-model.md, contracts/documentation.md, quickstart.md, tasks.md, analysis.md, verification.md and convergence.md in this directory.

Source ownership:

- CI researcher: internal/catalog/security_test.go, active CI/release workflows, their validation tests, current governance/development/product CI policy. No production lease behavior change unless demonstrated necessary.
- Routing researcher: site/lib/docs.mjs, static latest/current/version archive routes, postbuild/offline/export checks, routing/renderer tests and site/README.md.
- Documentation researcher: shared DocumentationFrame, article/reference scaffolds, site/app/styles.css, root layout/header and landing links, local site-runtime.js, bounded headless UI checker and presentation tests.
- Root: README attribution, CHANGELOG.md, slice artifacts, cross-agent integration, current application capture refresh, Project/PR/reviews/CI handoff.

**Structure Decision**: Reuse the existing renderer/frame/static exports and explicit controlled fixtures. No new runtime server, redirect service, application database option or package platform is added.

## Research and Decisions

2026-10-14: Spec Kit plan workflow dispatched three research agents for the actual lease/CI budget, static current-version routing, and whole-site documentation standards. Read-only reports resolve the design before implementation.

1. Real ClaimLease uses backend time and a1ms..1minute TTL. Tests seed a clearly live exact durable fixture, assert actual conflict without mutation, deliberately expire that same owned/generation row and exercise real takeover/fencing/renewal. Guarded fixture edits avoid wall-clock sleeps and do not change production APIs.
2. Render full current article bodies at stable /docs/latest/ and topical aliases, with canonical fixed-version metadata. Full static rendering preserves fragments and no-JavaScript reading. /docs/ becomes current-first; /docs/versions/ exposes archives. Explicit /docs/vVERSION routes remain immutable. Historical-only alias topics show a truthful current-index fallback.
3. Use a shared DocumentationFrame with native details disclosures for sidebar/contents; defaults remain compact without JavaScript, and desktop enhancement can expand them. Keep semantic navigation, skip target focus, current/version context, focusable locally contained complex content and hidden-until-functional theme controls.
4. Use existing Playwright/Chromium, fresh owned headless profile/context and exact cleanup for automated structure/overflow, keyboard, themes, zoom, file export and screenshot evidence. Automated results do not certify all assistive technology.
5. The owner already authorizes branch push and official PR. Stop only after exact final-head green CI and resolved findings (at most two requested rounds), for owner final review/merge. #97 is excluded.

## Verification Order

Run targeted regressions before changed behavior, then relevant Go/site/renderer/CI-policy tests, configured npm checks/tests, full affected site/public+offline builds and browser standards checks. Integrate agents only after blocking analyze passes. Commit clean code before refreshing provenance-sensitive application figures, then commit capture/evidence updates. Push and open official PR, attach it, count review rounds, fix findings and validate exact final head. Record all final gates and elapsed timing on the PR without falsely treating a pre-CI source snapshot as live completion.

## Complexity Tracking

No justified constitution violation is required. The owner-approved CI policy amendment is explicit, not an exception hidden in tests.
