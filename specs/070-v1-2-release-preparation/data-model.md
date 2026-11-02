# S070 data model

No application data model or catalog schema migration is introduced.

| Entity | Identity and relationships | Validation/state |
| --- | --- | --- |
| Candidate | version 1.2.0, exact revision, package and docs/schema inventory | Unpublished; all owned versions equal; four platform/variant combinations plus MSI/setup |
| Historical release | tag v1.1.0, original docs/schema/navigation/captures | Immutable original bytes; changes copied to new version first |
| Capture | original revision/version, source-file digests, renderer/fixture, image inventory | Clean source; isolated fixture; no private inputs; original image digests/dimensions |
| Qualification | workflow/attempt, source revision, verified readiness, terminal jobs, artifact digests | Mandatory checks passed, hard limit 3600 seconds; target 1800 advisory; reserve 60 provisional |
| Review handoff | PR/head, rounds and finding resolutions, retained audit digests | At most two review rounds; no unresolved actionable findings; owner controls merge |

Candidate states are prepared -> bootstrap-qualified -> captures-imported -> final-qualified -> owner handoff. Publication is outside this state sequence. Changes to the final source invalidate any claim that earlier head evidence qualifies that source.
