# Validation guide

1. Run npm run check and npm test through the verified hidden Windows runner.
2. Run Go tests/vet, Python unittest discovery and focused backup integrity/retention suites; use the existing alternative-backend fixture environment for S3/PostgreSQL/ArcadeDB.
3. Back up canonical audio/current transcript/model lineage, disable the source store, restore into an empty destination and compare all current authority digests and exact metadata/timing.
4. Corrupt or remove a bundle object and repeat restore: expect failure and unchanged target. Test reference-only cleanup fencing and explicit release.
5. Run native media preparation and package qualification on each supported operating system; assemble CLI-only and GUI-with-CLI variants and validate corresponding sources, relocation and final hashes.
6. Validate release candidate fixtures for version/source/asset mismatches and interrupted draft publication. No public release is created during this slice.
7. Run frontend/site checks, full CI on the final branch head and both bounded external review rounds; resolve every finding before owner merge handoff.
