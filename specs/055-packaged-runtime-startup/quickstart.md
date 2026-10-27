# Validation guide

1. Use pinned Go 1.27.1 and existing verified dependencies. Keep local children hidden and noninteractive.
2. Run focused owner-startup tests before the repair and record the old launch attempt under a held lock; run again after repair, including shutdown/restart, protected-input and deadline tests.
3. Run terminal evidence and terminal campaign Python tests, including private sentinels, wrong PID/stale records, malformed/oversized records, natural exit observation and cleanup preservation.
4. Run npm run check, npm test, go test ./... and go vet ./... through the hidden launcher. Run affected product checks without local native UI/installer launches.
5. Publish the feature PR and monitor exact-head applicable CI. Require all eight terminal cases in Windows/Linux source/package receipts. Audit each failed attempt before a full rerun.
6. Resolve at most two Codex rounds; verify the final head and all review threads, then hand off without merging. Report preparation, qualification and combined timing separately.
