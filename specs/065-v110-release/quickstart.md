# S065 validation and post-merge continuation

## PR phase

1. Run maintained preparation/history regressions, add the failing current-capability regression and make the proportional metadata correction.
2. Prepare 1.1.0 using `python scripts/release.py prepare --version 1.1.0 --date 2026-10-31 --highlights specs/065-v110-release/highlights.md`. Copy current corrections forward before restoring historical docs/schemas from v1.0.3; mark that historical entry released.
3. Omit only new-version screenshot walkthrough blocks, run root/local checks, and commit/push the clean capture bootstrap. Dispatch `gh workflow run ci.yml --repo shruggietech/insonic --ref codex/065-v110-release -f cold_media_source=false -f site_smoke=false`. Retain terminal acceptance and original capture artifact identities before another full run.
4. Verify and import only original receipt-enumerated images/receipts into docs/screenshots/v1.1.0. Restore new-version embeds, validate provenance and visually inspect the captures. Complete public/offline documentation and full maintained headless checks.
5. Converge, push and open the official PR. Resolve at most two review rounds and every actionable thread. Require green final PR and direct branch-head CI with retained terminal audits, clean source and correct tracking. Stop for owner merge.

## After owner merge

1. Confirm the reviewed merged-main SHA and completed exact-main CI. Dispatch the maintained Release candidate and publication workflow for that full revision with `publish=false` and `promote_docs=false`; this prepares publication-target bytes without publishing.
2. Retain the entire candidate, manifests, original receipts and checksums outside workflow expiry. Elect real-model CPU/NVIDIA acceptance separately if required for the publication campaign, using those exact package bytes and versioned corpus/model configuration. Do not represent deterministic fixtures as acoustic or hardware accuracy.
3. Under publication authorization, create the immutable v1.1.0 tag at the accepted source and use trusted-main `scripts/release.py publish` with the retained accepted candidate, exact source checkout and this slice's highlights. Do not rebuild after acceptance.
4. Promote the same published documentation with `pages-promote`, then execute `verify-public` against the original candidate, exact source and returned Pages commit. Verify all official asset digests, download identities and live documentation. Report phase timing and unverified dimensions separately.

## Command contract for the later publication campaign

Run Git/GitHub commands directly through the headless command runner. Run Python commands through a hidden noninteractive launcher with redirected output on Windows. The named variables below are campaign inputs, not values to copy from the pre-merge PR: `ACCEPTED_SHA` is the reviewed squash-merge source; `SOURCE_ROOT` is its clean checkout; `CANDIDATE` is the retained, digest-verified publication-target candidate; `EVIDENCE` is a durable receipt directory. Trusted-main tooling and the exact source checkout must be kept distinct when they differ. Supply `GH_TOKEN` privately; never include it in command logs.

Dispatch fresh preparation without publication:

```text
gh workflow run release.yml --repo shruggietech/insonic --ref main -f revision=ACCEPTED_SHA -f publish=false -f promote_docs=false -f documentation-transport=pages
```

Require terminal success for all elected jobs. Retain original run/job/artifact metadata, download the unique `release-candidate-ATTEMPT` archive, verify its GitHub SHA-256 digest, extract bounded regular files without path traversal, and validate its complete package/installer/documentation inventory using trusted `release.load_candidate(Path(CANDIDATE), source_root=Path(SOURCE_ROOT))`. Run that loader from trusted-main tooling with the selected paths:

```text
python -c "import sys; from pathlib import Path; sys.path.insert(0, 'scripts'); import release; candidate, assets = release.load_candidate(Path(sys.argv[1]), source_root=Path(sys.argv[2])); print(candidate['revision'], len(assets))" CANDIDATE SOURCE_ROOT
```

Retain original manifests, checksums and package bytes. The release workflow has no CI shared-readiness aggregate: report its observed preparation, qualification and combined intervals separately rather than applying an invented readiness timestamp or reusing a different run's CI gate.

If separately elected, the maintained real CPU installed-product journey uses the matching package receipt selected from the candidate manifest:

```text
python scripts/qualify-journey.py --mode real --package PACKAGE_ARCHIVE --package-receipt PACKAGE_RECEIPT_JSON --revision ACCEPTED_SHA --reference-corpus tests/fixtures/acoustic/manifest.json --recognition-model RECOGNITION_MODEL --diarization-model DIARIZATION_MODEL --timeout 3600 --output EVIDENCE/cpu-journey.json
```

Run that journey from the exact accepted source checkout on the elected Windows/Linux CPU host with verified managed prerequisites and exact pinned models. It rejects generic CI execution. Its receipt distinguishes preparation, acoustic checks, installed CLI operations, backup/restore and cleanup. Optional `--managed-offline-assets` names a previously verified managed bundle; it is not an alternative model authority.

A separately elected NVIDIA processing check is narrower than the full installed-product journey:

```text
python scripts/qualify-processing.py --run-real-engines --tools MEDIA_TOOLS_JSON --recognition-python RECOGNITION_PYTHON --diarization-python DIARIZATION_PYTHON --recognition-model RECOGNITION_MODEL --diarization-model DIARIZATION_MODEL --device cuda --threads 2 --timeout 3600 --output EVIDENCE/cuda-processing.json
```

Do not count that processing receipt as installed backup/restore or GUI acceptance. Neither real-model command is elected or claimed by S065.

After explicit publication authorization and final acceptance, create the immutable tag at `ACCEPTED_SHA`, then publish the retained bytes through trusted-main tooling:

```text
git tag v1.1.0 ACCEPTED_SHA
git push origin refs/tags/v1.1.0
python scripts/release.py publish --candidate CANDIDATE --source-root SOURCE_ROOT --highlights specs/065-v110-release/highlights.md
python scripts/release.py pages-preflight
python scripts/release.py pages-promote --candidate CANDIDATE --source-root SOURCE_ROOT
python scripts/release.py verify-public --candidate CANDIDATE --source-root SOURCE_ROOT --site https://insonic.io/ --pages-commit RETURNED_PAGES_COMMIT --output EVIDENCE/public-verification.json
```

Record the exact returned Pages commit, require its deployment to finish before public verification, and retain the public asset/route inventory. Do not rerun preparation or rebuild the accepted packages to publish them. Existing matching published bytes may be reconciled by the maintained tool; conflicting remote bytes are a failure. These commands document a later campaign and do not authorize or perform publication during S065.

Capture identities and qualification outcomes are recorded in [verification.md](verification.md); final exact-head receipts are retained outside the source and attached to the PR conversation.
