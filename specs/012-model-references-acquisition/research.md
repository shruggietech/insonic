# S012 Research

## Phase 0 (2026-10-09)

The installed speckit-plan skill dispatched three read-only agents covering catalog/model identity, durable jobs/import and CLI/desktop/provider contracts. All findings are resolved into D1-D10 in plan.md.

1. Decision: Reuse base installs and existing speaker families/versions/artifacts/associations. Rationale: catalog already owns exact identities and portable proof validation. Alternative: duplicate registry rejected. New domains require frozen v7 digest and schema/snapshot compatibility.
2. Decision: Shared immutable acquisition work and scheduler dependency eligibility. Rationale: model Execute CAS can race under independent downloads; four waiting parent workers would deadlock the acquisition queue. Alternative inline waiting rejected.
3. Decision: Freeze imports as well as processing. Rationale: import has a separate UUID-only diarization closure without expected digest, so processing-only changes leave retry drift.
4. Decision: Shared non-loading compatibility checks before acquisition and at execution. Rationale: existing supported role layouts are adapter-specific; capability alone does not prove loadable format. Legacy inference preserves accepted manifests.
5. Decision: Explicit manifest-catalog sources and custom manifests. Rationale: exact metadata/pinning can use existing transport while keeping custom-model choice unrestricted. Direct Hugging Face downloads need supported CDN/auth routing and SHA-256 for auxiliary files, which are not provided by Git blob IDs.

Primary provider evidence: [download/revision guidance](https://huggingface.co/docs/huggingface_hub/main/guides/download) and [HfApi metadata](https://huggingface.co/docs/huggingface_hub/package_reference/hf_api). These inform the provider contract without claiming direct HF support or running real downloads.

Key entry points: internal/models/manifest.go and service.go; internal/processing/adapters.go; internal/app/election.go, processing.go and work.go; internal/library/types.go and manifest.go; internal/pipeline/definition.go; internal/catalog/migration.go, records.go, snapshot.go, library_state.go and roster.go; cmd/insonic/domain.go; desktop/frontend/src/settings.tsx, screens.tsx and forms.ts.
