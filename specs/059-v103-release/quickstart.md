# S059 validation and post-merge continuation

## PR phase

1. Run maintained root check/test, full Python discovery, affected Go tests/vet and site checks/tests/build through verified hidden launchers.
2. Push the clean 1.0.3 capture bootstrap; dispatch CI on codex/059-v103-release with cold_media_source=false and site_smoke=false. Wait for exact source/capture outcomes.
3. Verify original artifact digests and capture source/version/hash/dimension authority, then import receipt-enumerated images and original receipts into docs/screenshots/v1.0.3. Restore current documentation embeds and run public/offline validation and visual inspection.
4. Push/open the ready PR, resolve at most two Codex review rounds and every arriving actionable external review. Require final exact-source CI and independent retained timing/receipt audit.
5. Verify clean source, tracking and owned cleanup; stop with the PR unmerged. Do not tag, publish or deploy.

## After owner merge

1. Confirm the reviewed main SHA and completed exact-main CI. Dispatch Release candidate and publication from main for that full revision with publish=false and promote_docs=false.
2. Retain its complete publication-target candidate, manifests, original receipts and asset bytes outside workflow expiry. Elect scripts/qualify-journey.py --mode real with --package (that exact Windows CLI archive), --package-receipt (its original receipt), --revision (that SHA), --reference-corpus tests/fixtures/acoustic/manifest.json, verified --recognition-model and --diarization-model inputs, --timeout 3600 and an owned output. Managed offline assets require their complete verified bundle. Require fresh destination processing/matching after restore, original-path withdrawal, provenance/integrity and successful cleanup. Report preparation, qualification and combined time separately.
3. Under publication authorization, create the immutable v1.0.3 tag at the reviewed revision. Use trusted main scripts/release.py publish with the retained accepted candidate, exact source checkout and this slice's highlights.md. Publish those original bytes; rebuilding can change archives and does not inherit CPU acceptance.
4. Promote the same published documentation with pages-promote, then run verify-public using that candidate, exact source and returned Pages commit. Verify original asset hashes, download identities and live content. Keep versioned observations and unverified dimensions under #147.
