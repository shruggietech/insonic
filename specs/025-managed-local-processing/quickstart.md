# Validation guide

1. Run root npm check/test, frontend check/test/build, Python tests and Go tests/vet using the installed verified hidden launcher on Windows.
2. From a relocated clean CLI/desktop package, initialize a disposable workspace and inspect workers catalog, models defaults and processing readiness. Acoustic tasks must identify missing prerequisites independently of media/subtitle readiness.
3. Save a managed recognition/diarization setup input from workers plan, including its observed expected_revision; submit workers setup and wait on its returned work ID. Cancel/retry and stale revisions must preserve accepted settings.
4. Preview/elect builtin exact model bundles, including license/access and storage destination. Reuse an explicitly selected compatible local model tree or acquire through ordinary model jobs. Repeat to establish verified reuse and progress.
5. Save a local pipeline with returned exact model IDs and inspect processing readiness from both clients. Inspect CPU and explicit unavailable CUDA fixtures; no frozen election changes.
6. Outside CI, deliberately qualify installed CPU profiles against exact local model snapshots through `scripts/qualify-processing.py --run-real-engines` and record measured versions/devices/resource/output bounds. Do not claim unexecuted GPU/platform combinations.
7. Compare final source/package identities, public/offline docs and capability matrix; publish the official PR under owner authorization, resolve every review thread within at most two rounds and verify green checks on its exact final head.
