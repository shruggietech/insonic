# S074 data model

## Contextual corpus

A bounded versioned manifest has a stable corpus ID, synthetic provenance, unique case IDs, unique scenario-local speaker keys, canonical names, voices and ordered text cues. Each voice has exactly one expected outcome and optional expected speaker key. Version 2 also requires an exact expected name for unregistered identities; all other name fields are null. Cases cover direct/two-voice introductions, duplicate names, quoted introductions, mentions, contradictory introductions, unregistered names and anonymous voices. Truth never enters provider input. Synthetic PCM is only an import carrier and does not represent the text acoustically.

## Measurements

A metric has integer numerator and denominator, plus value equal to their ratio or null for denominator zero. Per-case terminal state is completed or error; errors remain in the denominator/accounting inventory where applicable. Outcome counts distinguish supported-contextual, matched, unknown, ambiguous and conflicting. Identity precision uses accepted decisions; identifiable coverage uses known uniquely identifiable truth. Outcome correctness is separate from identity precision.

## Acoustic evaluation

Use the existing immutable acoustic manifest. Enrollment sources cannot equal held-out/unseen sources. Models bind exact base bundle identity, worker/interpreter/CLI hashes, explicit cpu/cuda policy and actual device execution attempts. Every held-out/unseen clip records expected reader, decision and score. Observations measure enrollment/matching with supplied voice intervals, not diarization error.

## Receipt and release lifecycle

Readiness/preparation precedes qualification. A receipt preserves source/corpus/model/configuration identities, timestamps, per-case outcomes/errors and limits. Failed campaigns remain failed and are retained before rerunning. A release candidate binds one clean revision and aligned version to immutable asset inventories. Final PR acceptance links exact branch and merge-candidate runs, original artifacts and all review dispositions; owner merge/publication are subsequent actions.
