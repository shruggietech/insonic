# Implementation Plan: S022 backend configuration and local query assistance

**Branch**: `codex/022-backend-service-setup` | **Date**: 2026-10-13 | **Spec**: [spec.md](spec.md)

## Summary

Implement #67 and #68 as one shared configuration/readiness slice: nonsecret complete backend plans, credential bootstrap, non-mutating preflight and explicit fresh provisioning, plus native Ollama discovery and query assistance with matching CLI/desktop forms.

## Technical Context

**Language/Version**: Go 1.27.1; existing React/TypeScript Wails frontend; Node/Python maintainer tools.

**Primary Dependencies**: Existing pgx, SQLite, MinIO, LadybugDB, ArcadeDB, native/vault/session secrets and Wails. No new engine/toolchain dependency required.

**Storage**: Existing six backend choices; generated IDs/revisions and immutable catalog profile history.

**Testing**: Go unit and integration fixtures, frontend contracts/build, Node schema/check suites, Python product qualification and exact-source CI.

**Target Platform**: Windows/Linux x86-64.

**Performance Goals**: Backend/service metadata aggregate timeout at most 30 seconds, 1 MiB bounded metadata, at most 16 endpoint candidates and 128 models; existing assistance/query budgets retained.

**Constraints**: Hidden noninteractive Windows launches; no preflight writes, secret serialization, silent route adoption, implicit populated-workspace migration, paid inference or attended release gate.

**Scale/Scope**: Two complete issues, current unreleased 1.0.1 contracts/docs; no release/tag/merge.

## Constitution Check

Pre-research and post-design pass: owner scope preserved; six supported backends retained; shared Go operations with CLI/desktop parity; credentials protected; configured routes persist; no new per-item review; public documentation separates implementation from shipment; hidden launches and exact-head CI remain required. No constitution amendments or exceptions.

## Project Structure

- `internal/backendsetup/`: plan construction, field validation, probe reporting and provisioning.
- `internal/catalog/`, `internal/artifact/`, `internal/graph/`, `internal/workspace/`: reusable adapter/transport/configuration helpers.
- `internal/assistance/`: native Ollama metadata/discovery/proposal protocol.
- `internal/app/`, `internal/contracts/`, `internal/runtime/`: shared service request dispatch and budgets.
- `cmd/insonic/`, `desktop/`, `desktop/frontend/src/`: pre-owner bootstrap and runtime operations/forms.
- `schemas/v1.0.1/`, `examples/`, `docs/v1.0.1/`: current contracts, executable guidance and help.
- `specs/022-backend-service-setup/`: internal decisions/tasks/evidence only.

## Design and decisions

Use staged identity/config publication for fresh setup. A failed plan/probe/provisioning cannot replace an existing active config. Existing configuration inspection and preflight remain available in Settings, while populated backend transfer uses established migration/backup paths. Credentials are entered separately through protected stdin/native input, assigned opaque references and resolved before catalog startup. Preserve explicitly chosen native/vault/session and S3 environment/anonymous authentication.

Network probes reject redirects, bound response sizes, close idle connections and use cancellation contexts. Probe dimensions report unknown/not-tested rather than pretending a path scan or server ping proves schema or writes. Plan fields have specific corrections.

Native Ollama metadata is inspected again before inference; local selection cannot silently invoke a cloud model. Missing capability or route evidence is surfaced with correction. Hosted requests do not assume structured-output support. Proposal parsing and graph query validation are unchanged authority.

## Verification and publication

Run required npm check/test and affected Go/vet, frontend/site, Python and backend qualification. Tests precede implementations for the new security/transport/configuration/protocol contracts. Use exact-head CI and respond to every review finding. Initial automatic reviews are round one; at most one additional Codex review request is round two. Stop for final owner review and merge.

## Complexity Tracking

No additional framework or architecture exception. Bootstrap exists because an unopened remote catalog cannot supply its own authentication/configuration workflow.
