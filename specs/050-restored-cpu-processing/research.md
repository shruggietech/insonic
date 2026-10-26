# S050 research

2026-10-26, read-only Spec Kit research agents and source review.

- Decision: use ordinary explicit generate/run and fresh work identities. Rationale: processing.go enqueues fresh requests; same-work replay is separate. Alternative force flag adds no necessary contract.
- Decision: preserve source/destination environment receipts separately and verify restored complete models. Alternative reacquisition obscures self-contained recovery.
- Decision: retain canonical media, profile/model outputs and completed process/train/match history. Public subtitles/storage contracts deliberately supersede current document/derivative rows; full alternate transcript retention is rejected.
- Decision: existing missing/changed portable pins, stable recovery successor, cancellation and probe failure tests supply the negative runtime contracts. Add focused evidence/lifecycle tests and run those suites; no static product defect was demonstrated.
- Decision: preserve old receipt interpretation, introduce strict fresh-destination evidence for new campaigns, and share resource/reference validators to prevent divergent acceptance logic.

Sources: scripts/journey_cpu.py; scripts/qualify-journey.py; internal/app/processing.go; internal/app/portable_work_test.go; internal/catalog/backup_processing_test.go; docs/v1.0.1/processing-setup.md; docs/v1.0.1/subtitles.md; docs/v1.0.1/storage.md. No external technology decision or new dependency is needed.
