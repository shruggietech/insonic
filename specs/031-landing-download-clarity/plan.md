# Implementation Plan: Landing-page download clarity

**Branch**: `codex/031-landing-download-clarity` | **Date**: 2026-10-15 | **Spec**: [spec.md](spec.md)

## Summary

Deliver #101 through the existing static Next.js landing page, governed CSS and isolated documentation verifier. Separate package groups and precise labels; keep one primary hero CTA. Factor a pure landing component so released and unpublished states can be rendered from the same product source for tests without publishing a release.

## Technical Context

**Language/Version**: JavaScript/JSX, Node 22+, Next.js 16.4.0, React 19.3.0.

**Dependencies**: Existing esbuild 0.28.2, React server renderer and Playwright core 1.62.1. No new dependency.

**Storage**: Static export; no new persistent product state.

**Testing**: Node test runner, site build/checkExport, existing headless Chromium harness on hosted Linux.

**Platforms**: Responsive public web; Windows/Linux x86-64 packages retain exact existing URLs. Offline help remains documentation-only.

**Performance**: Reuse the 180-second browser harness and 660-second CI budget. Existing docs and full native/package integrity gates remain elected normally.

## Constitution Check

Pass before research and after design: owner scope preserved; no runtime policy or backend changes; versions unchanged; authoritative docs/latest and archived identity preserved; kit bytes untouched; isolated headless verification; UTF-8/LF; public prose excludes planning; owner final merge boundary retained. No exception required.

## Research and decisions

See [research.md](research.md). Spec Kit plan research was delegated to two read-only agents for content/brand contracts and existing browser-harness behavior.

- Use explicit installer/portable groups and vertical spaced actions instead of adjacent inline anchors.
- Keep one filled hero link, outlined subordinate package actions, underlined alternatives/guidance with 44px minimum targets.
- Use an accurate Linux prerequisites/install anchor and remove the duplicate generic bottom packaging link. Windows guidance covers its separate installer and ZIP paths.
- Extract `site/app/components/LandingPage.jsx`; `site/app/page.jsx` supplies the real elected manifest/brand. A maintainer-only renderer compiles this same JSX with existing esbuild into an owned ignored cache and renders states with the same React installation. It cannot publish or change source status.
- Verify actual snapshot export plus an explicitly labelled released-state HTML fixture sharing the export's real styles/chrome. This avoids testing only unavailable download buttons in normal candidate CI.
- Extend the current isolated harness, with no Windows browser launch. Hosted Linux screenshots are downloaded and inspected. Keep network isolation, owned cleanup and security regressions.
- Custom UX checklist markers remain reviewer-owned (0/8); the built-in specification checklist is 10/10. Autopilot authorization permits implementation without an inter-step approval request; implementation evidence remains separate.
- Push and official PR are explicitly authorized. Round one opens automatically; request at most one second round after fixes. Stop for owner final review/merge. Live deployed inspection follows authorized deployment after merge.

## Project Structure

- `site/app/components/LandingPage.jsx`: complete landing content and ordered download groups.
- `site/app/page.jsx`: manifest/brand adapter.
- `site/app/styles.css`: token-based reflow/action hierarchy.
- `site/scripts/landing-fixture.mjs`: trusted-source test renderer, inventory contract and isolated fixture support.
- `site/tests/landing.test.mjs`: both publication states, five exact hrefs, standalone labels, grouping and all visible link contracts.
- `scripts/check-docs-ui.mjs`: whole-landing geometry, screenshots, input/focus and enlargement checks within existing lifecycle.
- `tests/docs-ui.test.mjs`: geometry and navigation-guard regression cases alongside existing isolation checks.
- `CHANGELOG.md`: concise behavior and dated tooling decision.

## Verification strategy

Run focused tests before implementation and require a meaningful red result. Run `npm run check`, `npm test`, site tests and full public/offline build locally through hidden redirected launchers. Run the extended actual browser cases on the hosted Linux CI runner; inspect full-page light/dark screenshots at 320/390/768/1440, 200% text and a 320px reflow equivalent to 400% from 1280px. Record actual Tab order/visible focus and intercepted native Enter/mouse/touch activation without downloading packages. Compare all five assets to the fresh official release API and original manifest whose digest matches that API. Never call those intercepted clicks a successful network download or claim screen-reader certification.

## Complexity Tracking

No constitution violations. One pure component and one maintainer-only renderer are proportional to publication-state and rendered-page acceptance.
