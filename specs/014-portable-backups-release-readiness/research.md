# Research decisions

## Portable authority

Decision: catalog-native capture/restore plus streamed digest-addressed directory bundles. Rationale: existing snapshot validation binds immutable journals to receipts; editing exported JSON breaks authority. Retain historical receipt identity required for provenance, copy only current/durable byte authority. Alternatives: raw catalog export lacks bytes; backend-native DB backups do not support cross-backend restores; tar extraction adds avoidable path risk.

Decision: stop local owner, acquire workspace lock and fence catalog writes during transfer. Rationale: app startup resumes work and a generic lease currently does not fence writes. Reference-only pins are persistent until explicit release. Alternatives: read-only best-effort snapshots race cleanup; implicit expiration can invalidate a backup.

Decision: reset graph checkpoints and rebuild accepted evidence. Rationale: source graph checkpoints cannot establish destination graph state. Alternatives: native graph file copies are backend-specific.

## Source-complete packages

Decision: owned native source builds, exact source archives/build recipes/configuration and capability proof. Rationale: Windows upstream bundles about 40 external libraries; Linux corresponding sources do not match the pinned binary. FFmpeg core alone is insufficient. Preserve built-in decoders and explicitly account for retained external input capabilities.

Primary references: [FFmpeg redistribution checklist](https://ffmpeg.org/legal.html), [upstream static-build source policy](https://github.com/eugeneware/ffmpeg-static/blob/master/README.md).

## Release and tracking

Decision: baseline 0.0.0 remains until explicit version preparation; update owned version bindings, preserve immutable prior docs/schema, never globally replace upstream pin versions. Decision: complete draft-first verified release upload, publication readback, then configured exact-version docs promotion. Rationale: interruptions must not publish partial assets or promote unmatched docs.

Primary reference: [GitHub managing releases](https://docs.github.com/en/repositories/releasing-projects-on-github/managing-releases-in-a-repository).

Decision: canonical labels validate before API mutations; preserve existing issue body/title/state and project fields. An opened-only classifier maps allowlisted area selections to labels without changing edited historical issues. Compatibility labels remain.

## Integration decisions and dependency disposition

Decision: keep destination credential selection and vault/native credential namespace independent of the imported catalog workspace UUID. The optional namespace retains existing credentials without copying secret bytes into the bundle or rebinding the destination vault.

Decision: transform nonportable acquisition payloads only in a disposable validated catalog, then admit new proof receipts. The source catalog remains unchanged. Restored incomplete local acquisitions require a fresh source election; original local paths cannot become resumable work. Both captured-source and generated-portable catalog proofs are recorded.

Decision: keep a committed restore passive with a durable pending receipt until exact destination configuration activation completes. Expiring transfer ownership alone cannot make a failed restore writable. Recovery binds the original bundle and destination profiles.

Dependency PRs #17 and #23 are integrated into this slice: setup-go 7.0.0, Wails 2.16.0, x/crypto 0.57.0 and x/net 0.59.0. Full local Go/frontend checks pass; the shared all-platform native CI boundary qualifies the resulting combination. Separate PR merges are unnecessary after S014 reaches main.

Decision: share one reusable native source workflow between CI and release, with dependency and FFmpeg compilation stages. Compact same-run transfers prove exact revision, compiler, recipe, dependency install files, binary hashes and original archives before consumers reuse them. This preserves full decoder input support and removes recompilation from backend/native qualification. Measure actual workflow turnaround as well as per-job timeouts.

Decision: resolve bare Windows build executables against the selected child PATH before hidden process creation. Windows process startup otherwise searches the parent environment, which can bind a cache identity to a different compiler than the configured source toolchain.

Decision: use independent same-platform source chains and verify the complete cache before scheduling cold dependency/FFmpeg stages. Warm paths publish a hash-verified same-run transfer immediately; cold paths retain both source stages. Local cold compilation takes 504 seconds before downstream native checks, so full cold CI turnaround cannot be claimed compliant from per-job limits. Live cold/cached timing remains an explicit convergence task.
