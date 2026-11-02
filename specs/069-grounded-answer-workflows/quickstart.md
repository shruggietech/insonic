# Validation quickstart: Grounded answer workflows

1. Prepare a clean synthetic workspace with maintained companion tools and choose either the deterministic fixture provider or an explicitly elected real model. Never use private media for this corpus.
2. Run `npm run check`, `npm test`, affected Go tests including answers/assistance/catalog/app/pipeline/CLI/schema, and maintained Python tests through hidden noninteractive execution.
3. Run the maintained synthetic answer evaluation and inspect separate retrieval, independence, citation, quotation, support and uncertainty metrics. Expected deterministic thresholds are100% applicable coverage/validity/fidelity/uncertainty, exact independence and zero unsupported claims.
4. Run installed package CLI qualification on Windows/Linux default backends and the Linux PostgreSQL/S3/ArcadeDB campaign. Exercise request, wait, show/results, saved pipeline, partial evidence, provider failure, linked retry, cancellation, export and source replacement. Confirm no secrets or original paths in exports.
5. Run adversarial fixture cases for invented/foreign/stale/shifted/omitted citations, quote mismatch, malicious text, oversized output, finite repair, source mutation during publication and unknown publication recovery.
6. Build/test the documentation site and execute every emitted CLI help example with maintained fixture substitution. Check UTF-8/LF, generated schema/help consistency and source-file integrity.
7. If model evaluation is elected, run the same corpus through the installed CLI and selected saved configuration, retaining exact model/configuration/corpus identity and conservative rubric limitations. Otherwise record not elected; fixture success is not model-quality evidence.
8. Commit/push and open the authorized PR. Retain exact-head and PR CI terminal receipts, address all comments within two review rounds, and prepare the compatibility/readiness packet for owner review and merge. Do not merge or publish a release.
