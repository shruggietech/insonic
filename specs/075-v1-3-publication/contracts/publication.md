# Publication contract

The maintained interfaces are scripts/release.py, scripts/pages.py, scripts/public-verification.py, scripts/website.py and scripts/check-live-site-ui.mjs. No runtime API or package format changes are introduced.

1. Prepare release.yml with the exact main revision, publish=false, promote_docs=false and documentation-transport=pages. Require terminal successful mandatory lanes and retained original candidate.
2. Validate source/provenance, all ten files and both documentation archives with require_release=True. Qualify actual CPU/NVIDIA against the original package and receipt.
3. Create v1.3.0 only at the selected source after conflict checks. Publish the accepted directory through release.py publish; no rebuild or replacement of conflicting assets.
4. Use release.py published for a fresh independent download of every official file. Match names, sizes and SHA-256 to the accepted candidate.
5. Promote the original public archive with pages-promote to the existing destination. Require the exact staged/pushed/deployed identity; reconcile an uncertain outcome only with that identity.
6. Use verify-public, website.py verify-deployed and eight headless live viewport/theme cases to verify current and historical content, download targets and rendering.

All operational receipts remain outside workflow retention. Public PR bodies and evidence are screened for executable review mentions; automatic review is round one and at most one deliberate follow-up is permitted. Final PR merge belongs to the owner.
