# S013 implementation and delivery evidence

2026-10-09, owner-authorized autopilot. Branch `codex/013-speaker-models-roster-matching`; scope completes #15 and remaining #35, advances #29. #15's historical S015 roadmap label does not change this slice identity. Squash merge remains owner-directed.

## Specification workflow

Active Spec Kit templates and feature identity were resolved through repository helpers. Specify, clarify, plan, tasks, analyze and implementation ran before convergence. Requirements checklist: 16/16. Analyze: 16 functional requirements covered, no contradictory or unmapped requirement. No extension hooks were present.

Convergence checked 16 functional requirements, 11 acceptance scenarios, four success criteria, seven recorded research/design decisions and five constitutional principles. Eight actionable implementation gaps were appended as T024-T031, implemented and verified. Final buildable-scope assessment has zero remaining missing, partial, contradictory or unrequested findings. SC-004's external delivery gates remain pending until the final PR head is green and reviews are satisfied.

## Implementation mapping

| Requirement | Implementation and meaningful evidence |
| --- | --- |
| FR-001/002 | `internal/voicemodels/preparation.go`, portable manifest builder and catalog current-reference APIs. A 105-reference fixture drains all pages; manual-only corpus excludes prediction feedback and includes source/document/mapping authority. |
| FR-003 | Canonical source-clock sample projection, visible selection/decode/signal/short-clip diagnostics and noncontent preparation receipts. Correction retires obsolete dataset/preparation/model manifests without retaining copied text/assignments. |
| FR-004/005 | `adapter.go`, `configuration.go`, `base.go`, `service.go` and `training_lifecycle_test.go`: pinned local protocol, explicitly elected hosted uploads/cancellation, exact saved profiles, checkpoint-only state and compatible resume. Built-in embedding enrollment is distinct from weight training. |
| FR-006 | Publication hash/size/shape/compatibility checks plus catalog work/current-corpus acceptance; source correction during adapter execution accepts no version and retires unaccepted objects. |
| FR-007 | Exact speaker output discovery/profile CAS and confined verified retrieval. Missing/corrupt bytes, malicious paths, false hashes, hosted-only handles and racing empty destinations leave no partial accepted destination. Origin remains unchanged through explicit reassociation. |
| FR-008/009 | Real offline pyannote embedding worker and bounded 64-clip batches; deterministic fixtures cover unknown, ambiguity including zero-margin ties, minimum evidence, absent/missing profiles and multiple voices. Quantitative insufficient evidence remains visible. |
| FR-010/011/012 | Thin frozen roster/profile/evidence digests, finite candidate fences, manual-preserving mapping acceptance and automatic-result supersession. Correction scrubs obsolete matching payload/results, retains attested request replay, rejects changed intent and never reruns recognition/diarization. |
| FR-013 | Shared app admission/scheduler, CLI commands and desktop Speakers/Models/Library controls. End-to-end runtime fixture covers training, profile selection, matching, manual correction, exact fetch, replacement and original-request replay. Late response regressions prevent old-job result mixing. |
| FR-014 | Catalog schema 9 and frozen populated schema-8 migration, portable receipt/state/association/member proofs, SQLite/PostgreSQL shared lifecycle tests and graph current-profile edges. Configured artifact/graph adapters retain shared contracts; actual alternative service qualification is delegated to PR CI. |
| FR-015/016 | Deterministic tests, CI engine guards, authoritative docs/help/glossary, versioned JSON contracts and dated changelog decisions. No real weights or acoustic models in required checks. |

Task touchpoints named `dataset.go`, `training.go`, `embedding.go`, `adapters.go` or standalone retrieval tests map to the final service files `preparation.go`, `service.go`, `embeddings.go`, `adapter.go`, `training_lifecycle_test.go` and `retrieval_failures_test.go`; these preserve the original behavior requirements.

## Local checks

- `npm run check`: text encoding/BOM/LF/mojibake, links, version navigation, kit integrity and JSON/schema checks pass. Registry: 21 schemas, 19 contracts, 26 examples at version 0.0.0.
- `npm test`: 65 tests pass, including invalidated output snapshots and transient-preparation contract coverage.
- Full `go test ./...` and `go vet ./...` pass. Final affected schema/contracts/voice-model/catalog/app rerun passes after convergence fixes.
- Race tests for catalog, voice-model and app packages pass (70.360 s, 52.835 s, 114.360 s respectively).
- Desktop: 72 rendered frontend tests, TypeScript check and build pass. Source bridge and hidden native WebView qualification pass.
- Documentation/site/offline help build: 27 exported documentation pages, relative links/anchors and generated contracts pass.
- Python: 20 repository tests and 15 deterministic processing-worker tests pass.
- Cross-process CLI qualification passes, including owner reuse, cancellation/retry/history, catalog transfer, artifacts, replacement, rosters and exact model references.
- Relocated Windows native package passes inventory, paths with spaces, CLI real media import, Cueson assembly/native export, roster/replacement, GUI bridge and hidden WebView checks with development companion paths isolated. Receipt is a dirty source-build qualification based on main `2ecb973`, not an official release artifact. Package qualification takes 139.094 s.

Real weight training, encoder execution, acoustic matching accuracy and engine performance were not measured. Generic local/hosted production transport and pinned pyannote execution paths are implemented; fixtures establish orchestration/integrity, not acoustic quality. Native Linux/macOS and PostgreSQL/S3/ArcadeDB service execution remain PR CI gates.

## Publication and reviews

Owner explicitly authorized push and official PR creation. Publish with closing references for #15 and #35; #29 remains an advanced coordinator. First automatic review is round one. At most one explicit second round is authorized. Reply to and resolve every actionable thread, confirm final-head green checks, then stop for owner final review and squash merge. No merge, tag or release is authorized here.

Publication/review receipts will be recorded once available.
