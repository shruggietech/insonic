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
- Desktop: 73 rendered frontend tests, TypeScript check and build pass. Source bridge and hidden native WebView qualification pass, including pending source-seek recovery across decoder readiness.
- Documentation/site/offline help build: 27 exported documentation pages, relative links/anchors and generated contracts pass.
- Python: 20 repository tests and 15 deterministic processing-worker tests pass.
- Cross-process CLI qualification passes, including owner reuse, cancellation/retry/history, catalog transfer, artifacts, replacement, rosters and exact model references.
- Relocated Windows native package passes inventory, paths with spaces, CLI real media import, Cueson assembly/native export, roster/replacement, GUI bridge and hidden WebView checks with development companion paths isolated. Receipt is a dirty source-build qualification based on main `2ecb973`, not an official release artifact. Package qualification takes 139.094 s.

Real weight training, encoder execution, acoustic matching accuracy and engine performance were not measured. Generic local/hosted production transport and pinned pyannote execution paths are implemented; fixtures establish orchestration/integrity, not acoustic quality. Native Linux/macOS and PostgreSQL/S3/ArcadeDB service execution remain PR CI gates.

## Publication and reviews

Owner explicitly authorized push and official PR creation. Publish with closing references for #15 and #35; #29 remains an advanced coordinator. First automatic review is round one. At most one explicit second round is authorized. Reply to and resolve every actionable thread, confirm final-head green checks, then stop for owner final review and squash merge. No merge, tag or release is authorized here.

Official [PR #39](https://github.com/shruggietech/insonic/pull/39) opened at implementation head `ad9694d2a85e3d1658144d5d34ce6c681b2dc230`, with M5 milestone and closing references for #15 and #35.

Round one completed with one P2 finding: an accepted dataset request with a nonempty recipe conflicted on replay after invalidation scrubbed that recipe. Creation receipts now preserve a canonical noncontent request digest and dataset identity; replay authenticates the original speaker/recipe election without resolving corrected evidence again. Changed speaker or recipe still conflicts. Catalog/service regressions cover original replay, changed intent, no mutation and portable restore.

Initial [CI run 37980169153](https://github.com/shruggietech/insonic/actions/runs/37980169153) exposed a PostgreSQL historical-fixture reset ordering failure. The schema-3 downgrade fixture now drops new dependent speaker tables before their parent tables, retaining foreign-key enforcement; its upgrade asserts schema 9 and recreated empty speaker authority tables. Targeted migration tests and integration-tag compilation pass locally; PostgreSQL execution remains the correction-head CI gate.

The authorized [second and final code/security review request](https://github.com/shruggietech/insonic/pull/39#issuecomment-6087906668) completed against `8f253b6231acc6480541602e92f3daaae558ea11`, reporting three actionable findings. Hosted response model/checkpoint artifacts now require inline bytes without paths, or an opaque hosted-only handle. Regression endpoints reproduce attempted prepared-audio publication and confirm no accepted model/checkpoint. Private adapter deadlines remain detectable across upload, response wait and body reads; declared cancellation uses an independent five-second control request, without claiming remote work stopped. Waiting-response and stalled-body fixtures prove cancellation can reach an independently running remote task after the training deadline.

Service and catalog share the 100,000-reference bound, with service overflow rejected before manifest publication. An actual 10,001-cue catalog corpus verifies admission, inspection and 100-reference continuation pages; transaction-local recording/cue/mapping caches remove repeated document decoding without changing authority. Empty/duplicate cue identities are rejected before an ambiguous index can authorize evidence. Full service creation/training performance for corpora of 10,001 or more references was not measured. If profiling finds remaining per-reference read/query cost material, a bounded catalog batch-resolution API is a proportional follow-up; no throughput claim or new authority API is introduced here.

No third review request is authorized or issued. These corrections must pass final-head CI before handoff.

After final-round corrections, full `go test ./...` and `go vet ./...`, `npm run check`, all 65 repository tests and the 27-page site/offline help build pass. Catalog and voice-model race suites pass at 115.888 s and 32.343 s; the rendered frontend has 73 passing tests and corrected source/native hidden-WebView qualification passes. Catalog/service fixtures verify all three final-round findings before publication.

[Correction-head CI run 37981202289](https://github.com/shruggietech/insonic/actions/runs/37981202289) passed documentation, repository/schema checks, Windows core, Linux native and actual PostgreSQL/S3/ArcadeDB fixtures. macOS relocated-package qualification exposed a readiness race that could lose an early source seek. Desktop playback now retries a pending elected seek at data readiness and clears it after confirmation so subsequent buffering cannot rewind ordinary playback. The rendered regression reproduces the lost seek and checks later user-position preservation; qualification targets, tolerances and timeouts remain unchanged. Corrected native package checks and final receipts remain pending.
