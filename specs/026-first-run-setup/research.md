# S026 Research

2026-10-13, Spec Kit Phase 0. Two research agents inspected persistence and desktop integration without modifying product code.

## Durable state

Decision: reuse `catalog.NamedSetting/PutNamedSetting` after provisioning, with typed setup data. Rationale: existing backend-neutral CAS, canonical request digest, replay before stale checks, portable setting/receipt export. Alternative: new tables or a separate host path file would expand migrations or lose portable state. Receipt revision identifies acceptance while current readback may be newer.

## Bootstrap

Decision: save a normalized backend plan with UUIDs before provisioning. Rationale: `backendsetup.Build` fills missing IDs and `ProvisionMode` retains exact provisioning intent; repeating `workspace.Init` cannot reconcile an unknown result. Alternative: create local defaults immediately prevents later fresh backend selection. Profile-private typed draft survives client restart; secrets remain separate.

## Existing effects

Decision: reuse BackendSetup, Workers, Assistance, Library and Jobs controls. Rationale: their shared operations already own routing, dependency preparation, protected credentials and work progress. Alternative: duplicating forms or orchestration engines would replicate CAS and validation logic. Save IDs/observations only after accepted effects.

## Accessible stages

Decision: ordered five-step progress, focusable step heading, explicit optional skip/defer controls, announced correction and no time limit. Rationale: [W3C multi-page guidance](https://www.w3.org/WAI/tutorials/forms/multi-page/) and [form notifications](https://www.w3.org/WAI/tutorials/forms/notifications/) support logical stages, progress and actionable errors; [Microsoft checklist](https://learn.microsoft.com/en-us/windows/apps/design/accessibility/accessibility-checklist) supports keyboard/focus/scaling checks. Alternative: a modal mandatory setup gate violates optional entry.

All technical unknowns resolved using current source. No product eligibility rule or extra approval is introduced.
