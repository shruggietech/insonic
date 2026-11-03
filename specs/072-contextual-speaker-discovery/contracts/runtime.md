# Runtime and provider contracts

- speakers.discover takes a recording item_id, elected assistance configuration, bounded max_cues/max_calls/max_observations and optional matching_work_id. It returns accepted durable work. Existing work show/results/wait/cancel/retry applies.
- speakers.hypotheses takes a recording item_id and returns the current typed aggregate or content-free absent/stale status. It never serves historical names after current authority changes.
- speakers.resolve takes a recording item_id, expected discovery revision and policy (minimum contextual roots, acoustic score and margin). It atomically publishes deterministic decisions after current-epoch/CAS validation. It does not assign mappings or enroll profiles.

All operations use the existing versioned runtime envelope and human/JSON CLI. Machine input/output schemas, help templates and desktop Operate dispatch must agree. Unknown fields and client-supplied acoustic scores/identity proofs are rejected.

The generated provider response is version 1, with an observations array. Each observation has a bounded name, optional aliases/role/relationship, evidence class, current voice/cue references, exact quoted text and uncertainty. Abstention is an empty array with explicit coverage. IDs/timing are validated against only the frozen selected current input. Any malformed observation rejects the complete response. Model output cannot add canonical speaker IDs, independent roots, acoustic evidence or accepted mappings.

Provider transport retains existing local/hosted route checks, secret references, redirect policy and bounded response handling. One parent configured deadline covers every call, cancellation and final publication. Extraction input excludes acoustic predictions to prevent circular confirmation.
