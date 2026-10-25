# S049 Validation

1. Install existing pinned root, frontend and site dependencies in prerequisite preparation.
2. Run `node --test tests/capture-application-ui.test.mjs tests/docs-ui.test.mjs`, then `npm run check` and `npm test`. Regressions must use nested temporary parents and preserve unrelated sentinels.
3. Build/restore clean exact-source native inputs and documentation exports using existing CI preparation. Run `python scripts/qualify-ui-capture.py --output build/ui-capture` on Windows and Linux. Run `node scripts/check-docs-ui.mjs` with the discovered system browser and exported public/offline help on both platforms.
4. Inspect source/binary receipt identity, fixture diagnostics showing actual canonical parent and allocated path, verified post-run absence, and unchanged isolation/functional checks. Reject successful receipts if cleanup fails. Retain remote artifacts, then remove only this slice's owned local scratch.
5. Require terminal applicable CI and elected acceptance with preparation/qualification/combined accounting, close every review thread from at most two Codex rounds, and hand the unmerged PR to the owner.

See [data-model.md](data-model.md) for lifecycle and forced-termination limits. No old host-directory sweep is part of these steps.
