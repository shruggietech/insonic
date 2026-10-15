# S028 research

2026-10-14, Spec Kit Phase0. All technical clarification items are resolved.

| Decision | Rationale | Alternatives considered |
| --- | --- | --- |
| Controlled exact durable lease fixture, real Claim/Renew operations | The100ms live lease can expire during slow Windows durable I/O. Explicit live/expired fixture states preserve exclusion/generation semantics on SQLite/PostgreSQL. ProductionTTLbounds remain1ms..1minute. | Longer sleep remains scheduling-dependent; changing production clocks/APIs is unnecessary; removing assertions is unacceptable. |
| Eleven-minute CI budget | Owner explicitly adds60seconds. Update relevant jobs/outer qualification/current governance consistently while retaining unrelated compile/application bounds. | Keeping contradictory ten-minute policy or hiding elapsed time fails #93. |
| Fully rendered static latest aliases | Supports Pages/file exports and arbitrary fragment anchors even with JavaScript disabled; manifest chooses latest and truthfully labels candidate/released. | Server redirects require a server; client-only redirects cannot preserve arbitrary no-JS fragment reading; global redirects destroy historical authority. |
| Shared semantic frame/disclosures and contained scroll regions | Corrects the mobile cloud, missing reference navigation and disappearing contents with portable progressive behavior. | Smaller links alone do not satisfy the standards pass; ARIA menubars add inappropriate interaction requirements. |
| Isolated headless evidence | Reproducible automation checks layout/keyboard/theme/file behavior without owner desktop disruption. | Owner-assisted captures and personal browser/window automation are excluded. |

Sources: current main CI38633239324 and lease security suite; current source docs renderer/export/runtime; [WAI disclosure](https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/), [reflow](https://www.w3.org/WAI/WCAG22/Understanding/reflow.html), [target sizing](https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum), [focus not obscured](https://www.w3.org/WAI/WCAG22/Understanding/focus-not-obscured-minimum), [Actions job limits](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idtimeout-minutes). Governed brand consumer contract, implementation guidance, recipes and semantic tokens remain local authority.
