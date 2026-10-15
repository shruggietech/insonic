# S029 publication contract

Pre-merge: official preparation PR references #97 without a closing keyword. Owner performs final review/squash merge after green exact-head checks and resolved findings; automatic open review plus at most one manual request exhausts the two-round cap.

Post-merge: select the reviewed squash-main full SHA and verify ancestry/checks. Preserve any existing immutable tag identity; reject conflicts. Create/push v1.0.1 only at the selected source. Dispatch existing release.yml from main with revision, publish=true, promote_docs=true, documentation-transport=pages and reviewed highlights.

The release workflow must qualify released public/offline docs (site tests, actual diagrams and headless UI), all four native archives and MSI before collecting the complete candidate. Existing trusted-main publisher verifies tag/source, paginated source checks, complete draft assets and hashes before publication. Pages receives only the original verified published archive.

Expected public assets: four Windows/Linux CLI/Desktop archives, one Windows MSI, documentation.tar.gz, offline-help.tar.gz, release-manifest.json and SHA256SUMS. Source/version, inventories, corresponding sources/notices, signing disposition and digests agree. No PR artifact relabeling.

Verify exact Pages destination commit and public documentation-build marker, released/current manifest, latest/fixed/archive routes, schemas and all release download URLs. #97 remains open for partial publication or failed deployment. Recovery retrieves verified original published assets and does not rebuild them.

Historical documentation/schema/download bytes remain immutable; current catalog labels may be corrected from verified public release facts. No new domain or external destination is introduced.
