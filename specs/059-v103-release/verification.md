# Verification: S059 v1.0.3 preparation

**Date**: 2026-10-29
**Status**: Prepared bootstrap; genuine captures, final CI/reviews and owner handoff remain pending.

## Performed work and checks

- Spec Kit specify/clarify/checklist/plan/tasks/analyze/implement ran through installed command skills and helpers. Ten requirements and four success criteria map to nineteen tasks; no critical/high analysis finding. Built-in quality:16/16; no custom reviewer-owned checklist or extension hook.
- Actual preparation regressions first demonstrated frozen-history replacement, duplicate final changelog links and stale owned download URL tags. Proportional repairs preserve historical test/fixture authority, distinguish dynamic current negatives, retain foreign upstream URLs and make the complete highlights suffix idempotent.
- Maintained prepare advanced current software/docs/schema to1.0.3 with published=false. The four changed1.0.2 pages were restored after copying corrections forward; the complete1.0.2 docs/schema trees match the published tag. Original assets/exemptions and model/tool identities remain unchanged; the worker catalog changes only its owned envelope.
- Root checks/schema validation pass; root tests219/219, site tests18/18, public/offline export169 pages, frontend type-check/build and tests109/109 pass. Full Python703 tests pass with two declared skips. Go vet passes. Earlier complete Go runs failed one protected-input concurrent cold-start test. Changes to temporary-directory and package concurrency did not resolve it reliably; the demonstrated Windows permission repair and its full-source revalidation are recorded below.
- An initial prepared-state suite exposed the stale website fixture and a mismatched host Go compiler. Corrected current URL bindings and selected the pinned compiler; focused setup tests7/7 pass. One terminal Ctrl+C probe failed once; focused terminal tests10/10 and subsequent full Python pass without terminal code/assertion changes. Its transient cause is not established.

## Delivery boundary

Issue196, milestone43 and Project7 track S059. Candidate capture sections are temporarily omitted only from new1.0.3 pages for a clean genuine capture producer. Existing historical captures and original receipts are not relabeled. Final delivery requires restored current images, source/hash/version validation, final exact-source CI, at most two Codex rounds, audited retained evidence and owned cleanup.

Owner merge, exact-main CPU/fresh-restored qualification, tag, product release, Pages promotion and public verification are later work described in quickstart.md. No owner-installed, physical-exhaustion, fresh acoustic or publication claim is made by this bootstrap.

## Demonstrated qualification repair

Repeated cold-start failures revealed unconditional Windows private-DACL rewrites at each runtime-directory lookup. A deterministic setter-spy regression failed with one unnecessary write on already-private authority. SecureDirectory now freshly validates exact current ownership/access and avoids that write; changed-ACL repair, write-failure refusal, original owner checks and the ten-second product startup limit are unchanged. All workspace security tests pass. Under the normal system temporary directory and CPU settings, three protected-input concurrent cold starts passed in0.37–0.70 seconds, with owner readiness16.7–240 ms and all four clients successful by393 ms. Complete normal-environment revalidation passes: go test -count=1 -timeout=30m ./... (all packages), go vet ./..., full Python discovery (703 tests, two declared skips), root check and root tests (219/219). No product deadline, CPU setting, temporary-directory override or assertion was loosened.
