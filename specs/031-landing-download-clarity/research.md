# Research: S031

## 2026-10-15: layout and link purpose

**Decision**: One primary hero jump, subordinate package actions and explicit independent installer/portable groups with governed gaps.

**Rationale**: Adjacent inline anchors in the current download cards have no action layout. Existing package URLs are correct; the defect concerns scanning, standalone labels and truthful guidance. [GOV.UK button guidance](https://design-system.service.gov.uk/components/button/) supports a clear main action and distinct secondary actions.

**Alternatives**: Restyling every link as a primary button would preserve competing hierarchy. Whole-card links would obscure independent options. Neither fits #101.

## 2026-10-15: reflow and targets

**Decision**: Measure the governed 44px target size, separation, label containment and whole-page overflow; inspect full-page images in both themes.

**Rationale**: [W3C target-size guidance](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html) distinguishes minimum target geometry/spacing from visual weight. [W3C reflow guidance](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html) uses a 320 CSS-pixel width for horizontal-language content at the corresponding zoom. Record 320px as a reflow equivalent, not a claim that browser zoom UI was operated.

**Alternatives**: A high device pixel ratio changes raster density rather than layout. Static source assertions cannot establish pointer geometry or visible focus.

## 2026-10-15: rendering and autonomous capture

**Decision**: Reuse the isolated Playwright harness on hosted Linux and render both publication states from the actual pure landing component using existing tooling.

**Rationale**: The normal export is candidate status; it cannot expose released download anchors. A released fixture must use the same product JSX and export styles/chrome, not a hand-built imitation. The existing harness already bounds its lifetime, blocks external HTTP/WebSockets and owns its profile/cleanup. Installed Playwright Windows helper cleanup lacks an explicit hidden-console option, so no local browser is launched.

**Alternatives**: A new local Windows browser launcher duplicates infrastructure and risks the owner's desktop. A second full production build would add cost and require restoring candidate artifacts. The small trusted-source renderer avoids both.

## 2026-10-15: official package identity

Fresh v1.0.1 release API lists all five package names and URLs. The original release manifest SHA-256 is `ce885efde09f7ba9d4b12d3b60ec9e87f918921e2db5fcc6ec2021fab743b8d8`, matching the fresh API digest. Its four package receipts and selectable Windows MSI agree on platform, amd64 architecture, variants and extensions. No package is rebuilt, republished or executed by this slice.
