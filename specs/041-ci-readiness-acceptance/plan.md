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
**Constraints**: No widened cache restore, cross-run relabelling, owner desktop execution or unrelated hosted settings mutation. The 2026-10-21 convergence authorizes only two-to-ten-second per-operation fixture calibration and existing PR publication; all outer deadlines remain fixed.
**Scale/Scope**: #161 readiness/enforcement plus the 2026-10-21 bounded fixture calibration in #154 context. A necessary targeted native behavior repair receives a separate dependent slice in the same PR.

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

## Authorized cold-source campaign (2026-10-20)

The owner authorized the cold-media-source campaign, main required-check activation, CI repair and evidence retention, stopping before PR #162 merge. Apply only the required-status-check update and compare the remaining protection fields by API readback. This has succeeded with GitHub Actions app 15368 and strict mode preserved.

Campaign question: can both platform source chains compile from absent compiled caches, reach verified readiness and finish all elected qualification within 660 seconds, followed by verified warm reuse of the same candidate? One variable: append a unique workflow-run namespace to compiled media cache addressing when an explicit workflow_dispatch Boolean is elected. Preserve original-source archive addressing, recipe/compiler/source identity and every transfer/native/product check. Retain the namespace across full rerun attempts, enabling a same-source warm comparison. Never delete shared or unrelated caches, widen restore keys, reuse cross-attempt artifacts or reset the clock. First-attempt campaign readiness must be cold; the successful full rerun must demonstrate warm readiness on both platforms. Normal PR CI also qualifies the final merge source.

Stop and retain evidence on any source, identity, transfer, native, election or timing failure; repair proportionally and rerun complete attempts. Evidence includes exact branch/merge source, event, run/attempt, cache keys/misses, producer logs, toolchain inventories, artifact IDs/digests, source mode outputs, gate receipt and terminal timestamps. Post-merge main evidence and closure of #161 remain outside the authorized stop.

Analysis gate: the suffix changes cache addressing only, with false preserving the ordinary key byte-for-byte. All nine functional requirements remain covered; no product, compiler, recipe, source, deadline or authorization change is introduced.

## Bounded Windows fixture convergence (2026-10-21)

The owner authorizes observations, one two-second baseline and one ten-second comparison, any directly justified fixture correction, final local/hosted qualification and publication on the existing branch/PR #162. Two hosted Codex rounds are already consumed. Stop before merge, release, branch deletion and issue closure. This amendment explicitly changes the previously fixed inner fixture allowance; the constitution and all outer deadlines are unchanged.

Use passive Win32 desktop/session/window observations with API errors represented explicitly. Observe BrowserVersionString once in the existing owning-thread environment callback and keep it distinct from available-loader lookup. Do not enable CDP navigation gating. Record successful UI posting/entry/return/completion and child stage times as well as failures; retain capture streams on successful children without changing their machine-readable action receipt.

Reuse the existing installer-invocation diagnostic workflow with one bounded window-calibration mode. Restore the exact current assembled delivery from an explicitly selected run/attempt/revision through the existing artifact-digest and package verifiers. Run the existing twelve-case packaged exit command twice sequentially on one runner and the same executable, with separate fixture roots reproducing equivalent cold/warm histories. Select only two or ten seconds through a validated qualification-only setting. Every arm retains its first failure and keeps the original 100-second matrix cap. The diagnostic never emits a candidate scope proof. Record immutable inputs, selected run/attempt, workflow/tooling source, runner image, package/executable hash, and all effective settings.

The two-second arm remains a diagnostic baseline even if it fails. A ten-second pass with joins above two seconds supports calibration; two sub-two-second passes are explicitly inconclusive and permit a documented ten-second policy decision. A later deadline or functional failure remains unresolved. No fixed sleep, deadline cascade, forced-kill success, synthetic keyboard substitute, unrelated dependency/runtime change or new attended gate is permitted.

Separate commits: this planning amendment, identically instrumented observations/calibration control, then any adopted allowance or demonstrated repair. Use local message-only native tests and mocked parsing/runner tests before the hosted cohort. A binding observation change is diagnostic only; a change to production readiness/input/release behavior requires the next available coherent dependent slice. Keep complete ordinary CI for the frozen final head and retain terminal receipts in artifacts/PR evidence without needless source commits. Do not repeat compiled-media campaigns unless preparation/cache/handoff/accounting or their applicable contract changes.

## Accepted fixture policy and timestamp authority (2026-10-21)

The completed bounded comparison passed both arms below two seconds and is causally inconclusive. Apply the authorized ten-second per-operation fixture policy, with all existing enclosing limits and concurrent archive workload intact. Retain the private validated two/ten selection only for the exact immutable replay; ordinary application startup never reads it. No production readiness/input/release behavior defect is demonstrated, so no dependent production slice is created.

First-attempt initial queue uses the identity-verified workflow creation endpoint alongside exact-attempt status/jobs; retain attempt-record creation as separate provenance. Terminal audits reject invalid ordering, incomplete or foreign records and unsuccessful elected results. Historical campaign measurements remain scoped to their original source and can be recomputed from retained timestamp authority; no product work moves into preparation and no timing limit changes. Freeze the final candidate before ordinary CI, retain terminal receipts externally, and reconcile T029 evidence through the PR/evidence channels without changing tested source merely to check its box. T016 remains unexecuted.
