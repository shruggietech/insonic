# Data model

## Contextual mapping lineage

Existing SpeakerMapping retains its stable recording/voice key, speaker ID, mapping revision and document digest. Origin adds contextual, distinct from manual and acoustic automatic. Server-generated provenance records version, evidence class, application operation, discovery work/revision/digest, exact recording/source authority, speaker identity revision, qualifying hypothesis/reference/root identities and policy. Client-supplied lineage is never accepted as authority. All references are bounded by existing discovery limits; no copied transcript text.

A current contextual mapping can enter corpus selection. An owner correction replaces it with manual origin. Retraction removes only the targeted contextual authority, preserving later/manual mappings. Contradictory reruns retire applicable previous contextual mappings. Unknown, ambiguous or unavailable new evidence preserves accepted mappings and supplies no new enrollment authority; unknown identities remain unknown unless independently registered through existing identity operations.

## Dataset and profile

CurrentReference retains exact source/document/mapping revision and gains explicit independent attribution provenance where required by the portable contract. Dataset/preparation manifests expose the actual evidence classes and lineage. Technical exclusions retain existing recipe and signal rules. Dataset invalidation retires derived current preparation while immutable completed artifacts remain fetchable. Profile current usability requires a current associated output/dataset; selecting historical bytes does not make stale evidence current.

## Workflow

A speakers.run work payload freezes recording/source authority, normalized discovery/provider configuration, application election, optional enrollment recipe/training election and exact pipeline/model/tool identity. Derived child IDs are stable under retry. Results retain accepted discovery/application/dataset/training/profile references and diagnostics, never copied cues or assignments. Running parents yield to pending while children execute; terminal cancellation prevents later child publication. Retry reuses accepted stages and current revision fences instead of rerunning accepted effects.
