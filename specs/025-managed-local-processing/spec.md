# Feature Specification: Managed local processing and task readiness

**Feature Branch**: `codex/025-managed-local-processing`

**Created**: 2026-10-13

**Status**: Ready for owner final review and squash merge

**Input**: S025 completes GitHub issue #69 under authorized autopilot, including automatic PR publication and at most two external review rounds.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Prepare usable local processing (Priority: P1)

An operator starting from an installed Windows or Linux package sees which local processing capabilities are usable, chooses compatible existing resources or a managed isolated environment, and obtains verified engines and models without supplying integrity hashes manually.

**Why this priority**: Media import currently works independently of the inference tools required for elected processing.

**Independent Test**: Start with a clean package and empty workspace, prepare an elected CPU processing environment and exact model bundles, and obtain actionable task readiness through either client.

**Acceptance Scenarios**:

1. **Given** a clean package, **When** the operator inspects processing resources, **Then** media companions are verified separately from missing engines and models, with specific available setup actions.
2. **Given** a compatible existing environment, **When** the operator elects reuse, **Then** versions and byte identities are calculated and verified before saved tools become available, without changing that environment.
3. **Given** no usable environment, **When** the operator elects managed setup, **Then** only the isolated destination changes, setup is supervised and noninteractive, and failed or cancelled setup does not replace working configuration.
4. **Given** a configured processing route, **When** the operator imports or runs elected work, **Then** it proceeds without an additional per-item review and uses its frozen resources.

### User Story 2 - Acquire and reuse exact models (Priority: P1)

An operator chooses a compatible model role and sees license, exact identity, total download bytes, required space and the elected storage destination before acquiring it through existing durable work.

**Why this priority**: Engine availability alone cannot make an elected stage usable.

**Independent Test**: Register and acquire a compatible bundle with multiple files, cancel/retry acquisition and reuse verified artifacts across consumers on filesystem and S3.

**Acceptance Scenarios**:

1. **Given** a supported model choice, **When** inspected, **Then** the operator sees its roles, immutable file identities, upstream access prerequisites and location, without loading its weights.
2. **Given** missing files, **When** acquired, **Then** existing shared acquisition work supplies progress and cancellation/retry; valid files and interrupted work are reused.
3. **Given** insufficient space, unavailable storage or damaged bytes, **When** preparing or inspecting, **Then** the affected prerequisite and corrective action are reported without calling the bundle usable.

### User Story 3 - Choose hardware and understand task readiness (Priority: P1)

An operator sees separate readiness for import, supplied subtitles, transcription, diarization and optional elected training, plus CPU recommendations and explicit GPU dependency facts.

**Why this priority**: A successful doctor/import check must not imply that inference can run.

**Independent Test**: Exercise missing, incompatible and available runtimes/models with CPU/GPU metadata fixtures; compare CLI and desktop responses.

**Acceptance Scenarios**:

1. **Given** only media companions, **When** readiness is requested, **Then** admission-only use remains possible while transcription and diarization identify missing prerequisites.
2. **Given** an elected GPU device with incomplete dependencies, **When** inspected or executed, **Then** the missing dependency and explicit CPU reconfiguration action are shown; the accepted job's device never changes silently.
3. **Given** complete verified prerequisites, **When** inspected, **Then** readiness identifies what was actually checked and distinguishes it from measured inference success.
4. **Given** local failure, **When** processing or setup fails, **Then** no hosted provider is substituted and prior accepted configuration remains usable.

### Edge Cases

- Offline setup with a verified local runtime/wheel bundle or reusable environment; unavailable network when acquisition is elected.
- Missing or broken executable, incompatible engine version, changed bytes after inspection, paths with spaces/non-ASCII characters and malformed model manifests.
- Cancelled setup, runtime restart, retry, concurrent edits or configuration revision conflicts, duplicate setup election and partial downloads.
- Insufficient disk space, inaccessible elected storage, S3 credentials unavailable and model access terms not yet satisfied.
- GPU hardware present but runtime/library support missing; CPU-only runtime; selected device fails after submission.
- Setup input credentials, provider responses and subprocess failures must not leak secret values to persisted work or diagnostics.
- A model registration, file verification and engine import are distinct states; no inspection loads arbitrary weights.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Inventory supported engine versions and model roles and expose the same processing operations through shared CLI/runtime and desktop contracts.
- **FR-002**: Inspect packaged tools, elected runtimes and managed models, distinguishing configured paths, verified files, available compatible engines, acquired models and a usable elected pipeline.
- **FR-003**: Calculate candidate executable/worker hashes and engine/runtime versions automatically; retain explicit overrides, exact identities and frozen job execution.
- **FR-004**: Offer managed isolated worker setup/acquisition and verified reuse of compatible existing resources without modifying arbitrary global installations.
- **FR-005**: Setup must have durable status, bounded supervised hidden child execution, progress, cancellation and retry; only successful verified setup may publish configuration.
- **FR-006**: Declare shipped worker assets and elected dependency identities in versioned capability/package records; preserve existing companion verification and historical inventories.
- **FR-007**: Provide supported compatible model choices with immutable identities, declared roles, license, exact download size and destination, without arbitrary model loading.
- **FR-008**: Model acquisition must reuse the existing durable acquisition/artifact system and elected filesystem/S3 storage, valid artifacts and interrupted work.
- **FR-009**: Readiness and acquisition must report insufficient space, unavailable storage, offline prerequisites and corrective actions without overstating availability.
- **FR-010**: Recommend CPU settings from observed capabilities and report explicit GPU hardware/runtime/library dependency state. GPU presence alone must not imply usable inference.
- **FR-011**: Selected device failure must explain corrective action and allow explicit CPU reconfiguration without changing an accepted job's frozen provider/device.
- **FR-012**: Report separate readiness for media import, supplied subtitles, transcription, diarization and optional elected training, identifying exact missing prerequisites.
- **FR-013**: Preserve admission-only use and configured processing/training without mandatory setup or per-item review.
- **FR-014**: Preserve existing route/model/pipeline elections and expected-revision updates; cancelled, broken or stale setup must not replace accepted tools/settings.
- **FR-015**: Keep credentials out of setup drafts, argv, persisted work, reports and logs; use protected channels and opaque references for elected upstream access.
- **FR-016**: Ordinary readiness must be bounded and perform no inference or model download. Acquisition/setup occurs only when explicitly elected; all unattended operations remain noninteractive.
- **FR-017**: Reconcile supported worker versions with upstream prerequisites and apply model access terms only for the corresponding elected upstream model, without new application eligibility restrictions.
- **FR-018**: Exercise meaningful failure/security/backend/client cases deterministically in ordinary CI, and retain opt-in measured real-engine qualification outside CI and merge requirements when supported combinations change.
- **FR-019**: Public documentation, schemas, commands, capability matrix and package records must describe actual release behavior, limitations and verification levels.

### Key Entities

- **Worker profile**: Supported operation roles, exact engine/runtime identities, device/dependency facts and installation state.
- **Setup election/work**: Nonsecret elected sources/destination, frozen dependency identities, revision expectations, durable progress/state and verified result.
- **Model choice**: Compatible role bundle, immutable manifest identity, license/access facts, total bytes and acquisition status.
- **Task readiness**: Task and elected resource identities, checked prerequisite states, missing corrections, chosen device and observed capability evidence.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Both clients expose a complete clean-installation path to elected local processing without requiring the operator to type a content hash.
- **SC-002**: Every readiness response separates all five task classes and names the correction for each missing elected prerequisite.
- **SC-003**: Cancel/retry, offline and incompatible-resource fixtures preserve accepted settings and never change the frozen device/provider.
- **SC-004**: Repeated setup/acquisition reuses verified resources and shared work rather than creating duplicate installations or downloads.
- **SC-005**: Security fixtures demonstrate zero secret-value persistence/diagnostic exposure and no mutation outside the elected managed destination.
- **SC-006**: Ordinary automated checks remain below the ten-minute turnaround target and perform no model inference or weight download.

## Assumptions

- S025 completes #69. The optional first-run wizard (#66), MSI (#65) and its screenshot guide (#64) remain later slices.
- Windows and Linux x86-64 remain supported. The existing selected storage/catalog/graph backends, model acquisition and pipeline contracts remain authoritative.
- Local processing is optional. Online downloads occur only on an explicit setup/acquisition election; offline reuse remains supported.
- Setup may offer separate recognition and diarization environments when upstream dependencies require it. A detected interpreter alone does not establish engine compatibility.
- Explicit maintainer real-engine qualification remains outside required CI; its evidence must name exact measured versions/models/device and must not imply untested GPU support.

## Clarifications

### Session 2026-10-13

- Managed setup and reuse are optional; existing configuration and admission-only behavior remain valid (FR-004, FR-013).
- Device selection is explicit and frozen for accepted work; an offered CPU correction applies only to a new configuration/election (FR-011).
- The setup wizard is out of scope; processing operations must nevertheless be usable directly in both clients (FR-001).
