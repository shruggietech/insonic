# S066 execution and verification

## Prerequisites and authority

Use reviewed product source `82a1bfd7e8f4e389d451789ec7840d874127f39c`, version 1.1.0 and completed main CI 41270520010. Commit S066 planning locally before switching the existing checkout to clean main for trusted commands. Preserve the separate S066 branch. If protected remote main changes, use separately pinned source and current trusted-main tooling; never silently change accepted product source.

Run Git/GitHub directly through the headless command runner. Run Python/Node through hidden noninteractive launchers. Supply GH_TOKEN privately from existing authentication in process memory, never in command text, logs or saved configuration. SOURCE_ROOT is the clean accepted product checkout; PREPARED_HIGHLIGHTS is its specs/065-v110-release/highlights.md. CHROMIUM_GUI_EXECUTABLE selects the installed full Chromium 1248 chrome.exe. CANDIDATE, DOWNLOADED, EVIDENCE and PAGES_TREE below are confined ignored directories; all command paths are selected explicitly. Use the matching pinned Go compiler for complete package validation.

## Qualify and retain

1. Run npm run check, npm test and existing release_test.py, pages_test.py and public_verification_test.py regressions in the foreground.
2. Commit planning and switch to clean main; verify current protected main and selected source identities, then run nonmutating trusted/Pages checks.
3. Dispatch the maintained preparation workflow:

```text
gh workflow run release.yml --repo shruggietech/insonic --ref main -f revision=82a1bfd7e8f4e389d451789ec7840d874127f39c -f publish=false -f promote_docs=false -f documentation-transport=pages
```

4. Watch the selected run through terminal success. Retain original complete run/job/artifact metadata and the unique release-candidate-ATTEMPT ZIP. Verify GitHub size/SHA256 before bounded regular-file extraction. Preserve workflow head and product input revision separately.
5. Load the complete candidate with trusted release.load_candidate using the clean accepted source, and separately run verify_documentation(require_release=True) on both documentation archives. Require exact version/source and ten assets. Read back retained bytes and record actual supported lane intervals plus total campaign elapsed time. Retain any failed campaign before a full rerun.

## Publish and promote exact bytes

Run from the clean trusted-main checkout after accepted candidate validation:

```text
python -B scripts/release.py validate --revision 82a1bfd7e8f4e389d451789ec7840d874127f39c
python -B scripts/release.py trusted --revision 82a1bfd7e8f4e389d451789ec7840d874127f39c
python -B scripts/release.py pages-preflight
git tag v1.1.0 82a1bfd7e8f4e389d451789ec7840d874127f39c
git push origin refs/tags/v1.1.0
python -B scripts/release.py publish --candidate CANDIDATE --source-root SOURCE_ROOT --highlights PREPARED_HIGHLIGHTS
python -B scripts/release.py pages-promote --candidate CANDIDATE --source-root SOURCE_ROOT
```

Check local/remote tag state before creation; an existing conflicting tag is a failure and must not be retargeted. Use the exact returned promoted Pages commit in all following verification. No preparation/build rerun replaces accepted publication assets.

## Independent public bytes and rendering

```text
python -B scripts/release.py published --revision 82a1bfd7e8f4e389d451789ec7840d874127f39c --source-root SOURCE_ROOT --output DOWNLOADED
python -B scripts/release.py verify-public --candidate CANDIDATE --source-root SOURCE_ROOT --site https://insonic.io/ --pages-commit PROMOTED_FULL_SHA --output EVIDENCE/public-verification.json
```

DOWNLOADED must not already exist. Compare all ten downloaded filename/size/SHA256 tuples with original CANDIDATE, including manifest and SHA256SUMS, and retain that distinct readback receipt. verify-public alone proves API asset metadata and live docs bytes, not repeat binary downloads.

Materialize the exact promoted Pages tree as data through a confined archive export, without running its contents or rebuilding documentation. Then run:

```text
python -B scripts/website.py verify-deployed --site-tree PAGES_TREE --pages-commit PROMOTED_FULL_SHA --output EVIDENCE/live-website.json
node scripts/check-live-site-ui.mjs --receipt EVIDENCE/live-website.json --output EVIDENCE/live-ui --browser CHROMIUM_GUI_EXECUTABLE
```

The full GUI-subsystem executable runs with the maintained headless=true mode; do not select the console headless-shell executable on this host. Keep parent launch hidden and I/O redirected. Require all eight viewport/theme cases and cleanup to pass; inspect resulting images. Report sampled render evidence separately from acoustic, native GUI and comprehensive accessibility claims. Actual maintained verification allowances are 600 seconds for live bytes and 900 for browser/cleanup; a separate follow-up tracks stale guide prose.

## Evidence PR

1. Return to codex/066-v110-publication; write internal verification, compact auditable receipts and the publication changelog entry. Preserve immutable release source and original historical assets.
2. Converge against requirements/decisions/constitution, rerun affected local record checks and commit/push the final evidence.
3. Open the official non-draft PR, attach it to the chat and update issue/project tracking. Automatic review counts as round one; at most one explicit second request is allowed. Resolve every actionable finding and thread.
4. Require green applicable checks on final PR head, retain terminal acceptance and review evidence externally, read back the final receipt and hand off for owner merge. Do not merge automatically.
