# Validation guide

1. Prepare existing repository prerequisites and deterministic provider/media fixtures; select S073 with .specify/feature.json. Read spec.md and contracts/runtime.md for the expected behavior.
2. Run focused Go tests for catalog contextual mapping, currentness, corpus enrollment, workflow and CLI. Cover direct self-identification, mention-only, duplicate names, circular match chains, mixed/overlapping/noisy/short/duplicate clips and explicit CPU/NVIDIA policy.
3. Run configured installed CLI discovery/application/enrollment. Inspect human and JSON results, cancel and restart work, replay request IDs, correct a voice during running work, retract a prior automatic decision and verify immutable outputs remain retrievable.
4. Run backend/portable tests for SQLite/PostgreSQL, filesystem/S3 and graph recovery. Inject lost commit responses, failed profile publication, outbox failure and cleanup interruption. Tampered contextual lineage must fail restore/current-state validation.
5. Run npm run check, npm test, go test/vet and affected site/frontend/package qualification. Distinguish deterministic protocol/device fixtures from elected real-engine acceptance; record exact environment and any unverified acoustic behavior.
6. Publish the branch and official PR under existing authorization, inspect CI and external reviews, resolve every finding within at most two review rounds and retain compact terminal receipts outside the runs. Leave final merge to the owner.
