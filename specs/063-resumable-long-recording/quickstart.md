# Validation guide: S063

1. Run `npm run check` and `npm test` for maintained contracts and repository checks.
2. Run affected Go tests for processing, process, catalog, backup, app, pipeline and CLI with the pinned toolchain; run Go vet on those packages.
3. Use deterministic native fixtures with elected pinned FFmpeg tools to compare window PCM against trusted full decode for compressed, offset, stereo and uneven-track media. Compare tenfold duration under the same window configuration.
4. Run deterministic word fixtures for repeats, pauses, multilingual punctuation, legitimate track overlap and missing timing.
5. Inject interruptions around artifact write/hash/catalog acceptance/final publication; retry and require zero repeated verified windows. Corrupt/substitute output and change source/model/configuration to prove refusal.
6. Run portable backup/restore and alternate catalog/storage fixtures; cancel and retry without deleting accepted source bytes.
7. Inject OOM and expired ownership/deadlines; require finite attempts and truthful actual-device evidence. Real CPU/NVIDIA inference is a separately elected maintainer check, never required CI inference.
8. Build documentation/frontend and execute affected site/native checks. Preserve exact-candidate receipts outside CI and hand off only after green CI and resolved reviews.
