# Validation guide

1. Run focused catalog and speaker-model fixture tests using the repository's hidden launcher. Expect complete corpus pagination, no automatic-feedback membership and stale input rejection.
2. Execute deterministic local/hosted training adapters against fixture audio. Inspect durable work, exact dataset/run/version origin, checkpoint compatibility and validated output artifacts. Cancel/retry and source-edit scenarios must not accept stale work.
3. List speaker models, inspect one exact version and fetch it into a temporary directory. Verify hashes and the portable manifest; hosted-only output must report unavailable downloadable weights.
4. Match a recording with fixture roster profiles and explicit threshold/margin/minimum evidence. Expect accepted, unknown, ambiguous and manual-preserved results without retranscription. Edit roster/profile/source/mapping during an attempt and expect conflict.
5. Run CLI/frontend/schema contract fixtures, portable SQLite/PostgreSQL/catalog/graph checks, `npm run check`, `npm test`, and affected desktop/site/product builds. Required checks must not initialize acoustic models.
6. Run converge, push and publish the official PR, resolve every external thread within two rounds, and verify green CI on the final head. Actual acoustic accuracy is separately qualified and not inferred from fixtures.
