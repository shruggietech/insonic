# S071 validation guide

1. Verify accepted source and green main CI; run npm run check, npm test and maintained release/Pages/public-verification/website regressions through hidden tooling.
2. Commit planning, switch to clean trusted main, retain/audit main acceptance and run read-only Pages preflight.
3. Dispatch release.yml with revision 61a5ba35e2b388965ac316412e5a380e6e99b258, publish=false, promote_docs=false and documentation-transport=pages. Require terminal mandatory jobs, retain attempt metadata/original ZIP and validate the complete candidate plus both release-target docs archives.
4. Recheck source/tag, create v1.2.0 at the selected source, publish retained candidate using specs/070-v1-2-release-preparation/highlights.md and promote its original documentation through pages-promote with an explicit receipt path.
5. Use published to download all ten assets freshly, compare original digests, run verify-public with exact deployed Pages commit and verify-deployed with exact deployed files; run all eight headless landing cases.
6. Record truthful evidence and CHANGELOG.md on S071, converge, pass affected local checks, push and open the official PR, resolve at most two rounds, retain final applicable CI and notify owner for merge.
