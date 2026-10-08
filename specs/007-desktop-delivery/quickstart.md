# S007 validation guide

1. Build docs/offline help, desktop frontend and native assets with pinned dependencies.
2. Run repository/schema checks, Node tests, frontend interaction/type checks and Go acceptance/vet.
3. Create a disposable workspace through the desktop. Import both committed audio/video fixtures and compare identities/metadata/dates through CLI.
4. Save a pipeline/person/alias/term, inspect effective context and correct a media date. Read each result through CLI. Use deterministic processing callbacks only in qualification.
5. Inspect current cues/mappings, play each fixture and seek a timed cue. Replace/correct results and prove old references fail. Try arbitrary/expired paths and relocation mismatch.
6. Start/cancel/retry controlled durable work. Close/reopen desktop and verify work persists. Exercise keyboard, focus, labels and theme/reduced motion.
7. Build each platform package, move/extract it to a path with spaces, clear development overrides and qualify packaged CLI/GUI/help/companions. Verify inventory and notices/source provenance.
8. Confirm every required CI gate uses zero transcription/diarization inference, model initialization and weight downloads. Official release publication is outside this guide.
