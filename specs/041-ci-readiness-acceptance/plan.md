# Implementation Plan: Verified CI readiness and elected acceptance

**Branch**: `codex/041-ci-readiness-acceptance` | **Date**: 2026-10-20 | **Spec**: [spec.md](spec.md)

## Summary

Use the existing verified same-run media pipeline as a separately measured preparation phase, with an explicit terminal warm/cold ready join. Gate all elected qualification, including foundation and docs, behind one readiness boundary. Add an always-running scope-aware aggregate, bounded metadata reads and a terminal timing audit mode. Start Windows setup after source/dependency readiness, with product work gated on successful foundation/docs within the original qualification deadline. Independent qualification jobs still run in parallel.

## Technical Context

**Language/Version**: Node 22+, Python 3.12, GitHub Actions YAML.
**Primary Dependencies**: Existing pinned cache/artifact actions and verified source pipeline; built-in fetch; existing YAML test parser.
**Storage**: Existing exact caches and same-run artifacts; bounded acceptance JSON/step summary.
**Testing**: Actual helper virtual-clock characterization, workflow orchestration simulation, aggregate failure matrix, API identity/timing cases, existing transfer regressions and root/site checks.
**Target Platform**: Windows and Linux hosted CI; local Windows headless verification.
**Project Type**: Internal build and acceptance tooling.
**Performance Goals**: Preparation reported separately; all elected qualification <=660 seconds; aggregate rejects at 645 seconds to retain 15 seconds for finalization, terminal timestamps audited afterward.
**Constraints**: No widened cache restore, cross-run relabelling, native deadline change, owner desktop execution, hosted settings mutation or publication in this local slice.
**Scale/Scope**: #161 and aggregate enforcement; #154 baseline/recovery follows this slice.

## Constitution Check

Owner explicitly approved separate preparation/qualification timing on 2026-10-20. Constitution 3.1.0 records that change before analysis. All elected checks start after the boundary; queue time within phases is counted. Existing package/provenance checks and native deadlines remain authoritative. Source/build/toolchain observation does not claim fully pinned system packages. No product or routing changes. Pre-research and post-design gates pass.

## Project Structure

- `.github/workflows/media-source.yml`: terminal ready join; package inventory observation.
- `.github/workflows/ci.yml`: preparation boundary, gated qualification and aggregate.
- `scripts/ci-acceptance.mjs`: election/result evaluation, bounded API identity/timing and terminal audit. Historical-source observation and ready checks use helpers from the exact defining workflow revision.
- `tests/ci-readiness.test.mjs`, `tests/ci-acceptance.test.mjs`, `tests/ci-scope.test.mjs`, `tests/release_test.py`: behavioral scheduling, refusal and workflow integration.
- `.github/bootstrap.json`: required aggregate declaration.
- `docs/v1.0.1/development.md`, `CHANGELOG.md`, constitution: authoritative contracts and dated decision.
- This feature directory: spec, research, data model, quickstart, tasks and verification. No external product interface is introduced, so no empty contracts directory.

## Chronological implementation sequence

1. Specify/clarify and record owner timing choice; research actual graph and source trust.
2. Add slow-producer/helper and orchestration regressions against the old graph; preserve failure evidence.
3. Implement terminal ready joins and phase boundary; retain verified transfers.
4. Add aggregate refusal/timing tests, then implementation and required-check declaration.
5. Record resolved toolchain packages without changing recipe/source acceptance.
6. Update contracts and root/site checks; inspect final diff and source integrity.
7. Commit locally. Hosted cold/warm qualification, required-check application and post-merge acceptance remain pending publication authorization, with issues open.

## Decisions and alternatives (2026-10-20)

- A single workflow with explicit phases keeps current-run transfers and avoids cross-run import complexity. A larger helper timeout would preserve the wrong graph.
- A terminal reusable ready job validates the mutually exclusive successful warm/cold producer. Skipped cold stages are expected only on the verified warm branch.
- Foundation/docs must wait for readiness so their runtime is included in qualification. The same helper also waits for foundation/docs, so retaining its speculative setup launch would preserve a second readiness assumption. Use explicit prerequisites for all consumed inputs; measure the resulting critical path before claiming timing compliance.
- The aggregate evaluates every top-level elected result and exact attempt metadata. The terminal audit reports max actual job completion, including finalization. Its preflight cannot promise provider finalization latency; it reserves visible margin and does not present a provisional receipt as final evidence.
- Live bootstrap application waits for publication of a real aggregate check. Applying required settings to unpublished local code would strand remote work.

## Complexity Tracking

No new service, paid runner, framework, product dependency or native policy. No constitution exception remains after the explicit timing amendment.

## Hosted convergence repair (2026-10-20)

The owner authorized PR publication and CI repair, stopping before merge. Run 39493137969 demonstrated successful warm source readiness but failed the qualification budget: Windows assembly spent 206 seconds in cold Go product building, interactive completion reached 666 seconds after readiness, and archive qualification hit its job timeout during the unchanged final idle allowance. A warm retry cannot establish a repair.

Add a Windows dependency preparation job alongside source preparation. Discover the existing package build contexts, compile only standard library and explicitly non-main-module libraries into an empty isolated Go build cache, and transfer it plus the verified module download cache through a digest-bound exact-source/run/attempt artifact. Preparation must never compile product packages, executables or tests. Match package variant CGO flags and resolved compiler/native-library identities before restoring. Preserve every consumer build/check and all native deadlines. Only Windows package consumers require this additional handoff; independent runtime lanes remain unchanged. Record failed and successful hosted timing without treating a rejected run as accepted.

Analysis gate: nine requirements remain covered; dependency-only preparation fits the owner-approved contract. No product work moves before readiness. No weaker integrity check, timeout increase or cache fallback is introduced.

## Second hosted timing convergence (2026-10-20)

Run 39497295880 at head a6e58fd41cf4a7559824abc62573c2ba0ea31467 (tested source 126ce06e9504b02738c516ef45e15908a3df67f0) passed every product check, all four Windows scopes, joined package proof and candidate verification. The elected aggregate rejected total qualification beyond 660 seconds. Assembly fell from 206 to 101 seconds, but its setup still restored an unused default Go cache for 26 seconds. The setup EXE uses a distinct CGO-disabled, trimpath compiler context not covered by the five prepared contexts. Archive passed at 21:17:37Z; join ended 21:18:21Z, candidate 21:19:21Z and aggregate 21:19:34Z.

Next controlled change: prepare the setup compiler dependency context (excluding product code), disable default Go cache restoration only where the verified package dependency cache replaces it, begin Windows consumer setup after global readiness while enforcing a separate foundation/docs gate bounded by the original readiness plus 660 seconds, and join independent module/source verification and npm installations in parallel. Product compilation starts only after gates; source transfer helper remains capped at 180 seconds. All native cases, complete idle allowances, source/compiler/digest checks and isolated setup source rebuilds remain unchanged. Analyze gate: no timing clock reset, product work before readiness, verifier isolation change or weakened acceptance.
