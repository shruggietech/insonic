# S050 validation

1. Use hidden noninteractive Node/Python/Go launchers and pinned dependencies. Run root npm checks/tests, affected Python receipt/lifecycle tests, existing Go portable worker/model/setup/cancellation suites and affected site checks.
2. Commit clean source and obtain its digest-verified complete Windows CLI archive and package receipt from CI. Verify the exact revision and unchanged harness; never substitute an old local executable.
3. Run scripts/qualify-journey.py with --mode real, --package, --package-receipt, --revision, --reference-corpus tests/fixtures/acoustic/manifest.json, verified read-only --recognition-model/--diarization-model inputs (add --managed-offline-assets only if a complete selected bundle exists), --timeout 3600 and an owned --output. Do not inject a CI environment marker.
4. Independently validate the complete receipt against the frozen corpus. Require original-path withdrawal, destination setup and fresh recognition/diarization/matching, preserved canonical/profile/history evidence and successful owned cleanup; record training readiness without overclaiming it.
5. Retain compact source-bound receipts/failure attribution on GitHub, remove owned scratch after verification, complete final-head elected CI and at most two review rounds, then hand off unmerged.
