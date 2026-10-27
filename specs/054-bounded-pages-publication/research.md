# Research: Bounded publication

## Observed failure

#189 records one 46,714,316-byte tree request rejected with 422. The original accepted 1.0.2 export was deployed by verified normal Git. Source repair is distinct from that completed publication.

## Primary references and decisions

- [GitHub tree API](https://docs.github.com/en/rest/git/trees): base_tree extends an existing staged tree; omitting it creates only elected entries. Recursive responses can truncate, so require complete inventory.
- [GitHub blob API](https://docs.github.com/en/rest/git/blobs): supports UTF-8/base64 blobs up to 100 MB. Existing accepted files are at most 16 MiB; bound their JSON below 24 MiB.
- Choose 1 MiB serialized tree batches conservatively below the observed failed payload, with standalone blobs for non-fitting entries. No claim is made that 1 MiB is a published server limit.
- A native Git transport is valid but introduces child/credential/checkout lifecycle work; bounded API staging is smaller in scope and preserves existing deployment code. Staging all files as individual blobs would multiply content-creation requests, so keep small text inline and reuse identical blob identities.
- Do not retry generic errors or rate limits automatically, renew deadlines or drop history to fit requests. Bounded diagnostics expose failure for appropriate operator action.
- [GitHub API best practices](https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api#pause-between-mutative-requests) recommends one second between mutative requests. Pace the 82 original-export staging writes inside the same deadline and use deterministic virtual time in regressions.
