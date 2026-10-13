# Implementation Plan: Optional, resumable first-run setup

**Branch**: `codex/026-first-run-setup` | **Date**: 2026-10-13 | **Spec**: [spec.md](spec.md)

## Summary

Deliver #66 as shared optional setup orchestration over the merged backend, worker, model, query-service and import operations. Persist a bounded normalized bootstrap draft before a library exists and typed catalog state after provisioning. Both clients use these contracts, with no global prompt or import gate.

## Technical Context

Language: existing Go 1.27.1, React/TypeScript, Node maintainer tooling and Python verification. Dependencies remain unchanged. Target: Windows/Linux x86-64 CLI and Wails desktop.

Storage: profile-private bootstrap file with a lock and atomic replacement; existing SQLite/PostgreSQL NamedSetting CAS for portable library journey state. No new SQL schema. The bootstrap retains normalized workspace/profile identities; catalog state contains bounded IDs/enums only, no host paths.

Testing: Go domain/CLI/native bridge tests, JSDOM UI journeys, schema checks, npm check/test, Python configured suite, frontend typecheck/build, public/offline site export. Native CI retains supported-backend/package lanes and exact-source verification. No new dependency download or model inference is needed to prove orchestration.

Constraints: noninteractive commands, hidden Windows child launch, bounded state and receipts, strict payloads, safe retry identity, protected credential channel, existing CAS effect authority and no implicit routing. Readiness probes use the existing bounded processing contract; ordinary state reads do not start workers or downloads.

## Constitution Check

Pre-design and post-design: PASS. CLI-first shared runtime; filesystem/SQLite/LadybugDB defaults and full supported alternatives; no per-item approvals; no secret drafts; unchanged data identities and configured authorization; internal planning excluded from public docs; version 1.0.1 remains aligned. MSI/release/merge remain outside S026. No exception.

## Project Structure

- `internal/firstsetup/`: bootstrap draft, location preview and shared typed journey input.
- `internal/app/setup.go`: catalog-backed state, CAS and fresh readiness/observation overlays.
- `cmd/insonic/setup.go`: location/bootstrap and workspace setup commands.
- `desktop/setup.go`: pre-workspace native wrapper.
- `desktop/frontend/src/setup.tsx`: accessible optional stages reusing existing forms.
- Existing contracts, runtime schema, command help, client bridge and app navigation receive narrow integration changes.
- `docs/v1.0.1/setup.md`, current desktop/CLI/index documentation and changelog describe executable behavior.

## Decisions and sequence

2026-10-13: Location and backend choice precede first provisioning. Existing backend provisioning cannot migrate a populated library. Retain normalized IDs in a profile-private draft and provision only through the existing retryable intent. Existing libraries show accepted roles without overwriting them.

2026-10-13: Keep portable post-provision journey state in NamedSetting. Return accepted receipt revision and current reconciled state separately; later changes never masquerade as the original response. No separate database or path-bearing portable plan.

2026-10-13: Setup saves metadata only. Worker/model/service/import effects retain their own stable identity and CAS. The UI reuses existing controls and reconciles state after effects; closing/defer does not cancel accepted work.

2026-10-13: Default location is the platform user data directory plus `libraries/My voice archive`, editable and previewed outside installation bytes. Capacity is observed at the nearest existing directory; growth depends on elected media/models, and unknown estimates are explicit.

## Implementation phases

1. Validate typed inputs and bootstrap persistence with restart/replay/conflict/security tests.
2. Add catalog CAS state, exact receipt reconciliation and observed stage proofs.
3. Integrate noninteractive CLI and desktop bridge; keep direct workspace/import routes intact.
4. Add five-stage accessible UI with saved drafts, optional forms, summary/readiness and existing import/jobs/settings navigation.
5. Update contracts/docs, verify all configured checks, converge, publish the official PR and resolve at most two review rounds.

## Complexity Tracking

No constitution violations or new service/dependency. Bootstrap storage exists only because the authoritative catalog does not yet exist.

2026-10-13: Native package smoke now exercises setup save/replay/reopen, independent readiness and ordinary fixture import after deferral. This extends existing package acceptance without a new CI lane. Nonsecret bootstrap credential-store selection persists alongside the normalized plan; protected values remain transient.
