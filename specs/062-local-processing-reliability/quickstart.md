# S062 validation guide

1. Prepare the repository's pinned Go, Python and Node tools using hidden noninteractive execution. Confirm supported native/backend prerequisites before qualification; record preparation time separately.
2. Run Python worker/probe regression tests, affected Go package tests and schema checks. Exercise auto/CUDA/CPU, legacy omitted devices, bounded fallback and hostile/malformed provenance.
3. Run speaker enrollment/matching fixtures with unchanged model/sample identities, exact silence/no-speech, quiet/nonzero disagreement, invalid-language admission and accepted-data preservation.
4. Run golden normal/verbose/debug/JSON output cases and artifact retention/lease/grace refusal tests. Run maintained documented workflow and operator checks against source and extracted Windows/Linux package CLIs.
5. Run npm run check, npm test, go test ./..., go vet ./..., affected frontend/site and supported backend checks. Retain terminal results and preparation/qualification/combined elapsed time under existing policy.
6. In a separately elected compatible local environment outside deterministic CI, run real CPU/NVIDIA recognition, diarization and enrollment/matching on owned synthetic/licensed fixtures. Record actual device/model identities and limitations. Never use the active desktop for GUI or attended acceptance.
7. Publish the authorized PR, inspect CI and all review channels, resolve findings, use at most one additional Codex review request, and retain a final exact-head handoff without merging.
