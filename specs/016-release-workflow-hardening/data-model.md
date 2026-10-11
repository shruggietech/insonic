# Validation entities and transitions

| Entity | Fields / relationships | Validation |
| --- | --- | --- |
| Candidate | version, clean revision, workflow attempt, six platform/variant package receipts, source hashes, public/offline archives, signing dispositions | Existing complete closure/source/inventory checks; release target required on release dispatch. |
| Pages destination | repository, legacy build type, gh-pages branch/root, insonic.io domain, current ref/tree, marker files, environment policy | Existing exact destination only; no settings mutation or unexpected alternate host. |
| Pages tree | verified archive members plus CNAME/.nojekyll, parent commit and release version/revision | Confined regular unique members, no .git/traversal/aliases; historical routes preserved; fast-forward only. |
| Pages result | requested build, staged commit, built/failed/queued result, bounded deadline | Success requires exact staged commit, not any previous successful build. |
| Package election | archive and verified receipt, exact revision/version/platform, extracted manifest/inventory/tool/worker hashes | Missing/tampered/stale identity rejected before application operations; no source/ambient fallback. |
| Journey result | selected package plus CLI/worker/tool/model/device hashes, owned transitions, accepted/recovered authority, limitations | Exact bytes/values preserved; unknown matching is valid diagnosed behavior. |

Transitions: candidate verification -> destination preflight -> publication authority verification -> confined tree preparation -> nonforced ref advancement -> explicit build request -> exact-commit completed build. Dry-run stops before destination mutation. Failure before validation performs no writes; race/build failures cannot report promotion success.
