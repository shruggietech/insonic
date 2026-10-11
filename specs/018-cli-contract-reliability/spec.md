# Feature Specification: S018 CLI contract reliability

**Feature Branch**: `codex/018-cli-contract-reliability`

**Created**: 2026-10-11

**Status**: Specified

**Input**: Complete issues #49, #50, #51, #52, #53, #54, #55, #59 and #60 as one CLI contract slice under autopilot, push an official PR, resolve all reviews with at most two rounds, and stop for owner merge review.

## User Scenarios & Testing

### User Story 1 - Discover and invoke commands predictably (Priority: P1)

Operators can obtain contextual help without opening a workspace, starting work or creating files, and can supply literal filenames/text without global flags consuming them.

**Independent Test**: Exercise every command/alias help path inside and outside a workspace, then invoke literal and option-value cases against disposable files.

**Acceptance Scenarios**:

1. Given no workspace, when any group/leaf receives `-h`, `--help` or `help COMMAND`, then useful exit-zero help includes syntax, examples, aliases, defaults, limits and required input templates without accessing files, providers or runtime state.
2. Given a literal delimiter or a value beginning with a global option token, when parsing the command, then data retains its identity and machine-mode error behavior is independent of option order.
3. Given malformed, duplicate, mutually exclusive or unknown arguments, when invoked, then a bounded option-specific correction is returned before workspace/dependency access; repeatable speaker selectors remain supported.

### User Story 2 - Perform existing operations through the CLI (Priority: P1)

Settings, paged speaker-model lists and vault credential deletion behave through the same authority as the runtime/desktop.

**Independent Test**: Read/write/reset settings with revision conflicts, compare both model-list spellings, and delete a synthetic vault entry across cold/live locked/unlocked owners.

**Acceptance Scenarios**:

1. Given current settings, when the CLI writes or resets tools/appearance, then shared state changes once; stale revisions and malformed values fail without mutation.
2. Given a speaker, when either list spelling requests the first or subsequent page, then equivalent valid pages are returned; invalid cursors and limits are rejected.
3. Given a vault credential, when delete receives correct, wrong or missing protected input across supported routes, then results agree, failure retains the entry, and no secret reaches arguments, outputs, logs or persisted plans.

### User Story 3 - Preserve date and response precision (Priority: P1)

Operators and machine consumers see exact stored integers and timestamp interpretation regardless of the host timezone.

**Independent Test**: Compare raw stored, local-protocol and CLI integer tokens and run the same explicit-offset imports on UTC and non-UTC hosts.

**Acceptance Scenarios**:

1. Given nested signed integers outside binary64 exact precision and at supported 64-bit boundaries, when rendered in human or machine mode, then decimal tokens retain their exact values and declared numeric types.
2. Given a timestamp with `Z` or a numeric offset and no elected timezone, when imported or corrected, then the same instant is selected on every host while the raw literal and zone provenance remain inspectable.
3. Given an explicitly contradictory zone, DST fold/gap, date-only or date range, when interpreted, then documented policy and uncertainty remain explicit without a new review gate.

### User Story 4 - Consume reliable results and durable failures (Priority: P1)

Machine consumers receive a documented result/error shape and exit behavior across direct, maintenance and live-runtime routes; failed durable work preserves its actual cause after restart.

**Independent Test**: Parse documented streams and schemas for every route, reload failed work, and compare show/results/wait classifications.

**Acceptance Scenarios**:

1. Given success, invalid input, missing workspace/provider/item, incompatible version, timeout, partial completion or failed work, when `--json` is used, then a versioned response retains correlation and useful result/error context with documented stream/exit behavior.
2. Given cold/live/expired credential ownership, when the same operation is invoked, then its public response shape remains equivalent.
3. Given conflict, checksum/limit/timeout/dependency failure, cancellation or mixed-item results, when work is persisted and the owner restarts, then bounded classified reasons remain inspectable without persisting arbitrary provider output or secrets.
4. Given an output encoding/write failure, when the CLI exits, then it reports failure rather than successful delivery.

### Edge Cases

- Help tokens after `--` and tokens used as flag values remain literal data.
- Help includes maintenance/internal commands without restricting access.
- Unknown commands/options fail before any workspace discovery or input-file read.
- Old generic work failures retain their original meaning; new typed causes do not reinterpret historical records.
- Retry fencing, immutable receipts, terminal state and partial item outcomes retain existing authority.
- Vault input remains bounded and cleared; native/session deletion remains input-free.
- Optional absent pagination fields remain absent; UUID validation remains strict.
- Redirected output contains no diagnostic chatter; output failures cannot recurse indefinitely.

## Requirements

### Functional Requirements

- **FR-001**: Honor global and command argument arity, literal termination, aliases, duplicate/mutual-exclusion rules and repeatable choices (#51).
- **FR-002**: Resolve complete contextual help and useful group invocation before access or mutation (#50).
- **FR-003**: Admit settings through shared operation authority and preserve revision/reset validation (#49).
- **FR-004**: Omit absent model cursors and preserve equivalent pagination/defaults (#55).
- **FR-005**: Read bounded vault-delete protected input once and forward identical bytes through each route, preserving clearing and failure retention (#54).
- **FR-006**: Preserve exact signed integer values through nested response decoding and rendering without globally stringifying numeric fields (#52).
- **FR-007**: Distinguish implicit host timezone from an explicit election; preserve raw observations, nanoseconds, fold/gap/date/range policy (#53).
- **FR-008**: Define and implement envelopes, streams, correlation IDs, help/version output and exits across all public routes (#59).
- **FR-009**: Persist bounded typed durable causes and safe diagnostic context through show/results/wait/reload, preserving old records and fencing (#60).
- **FR-010**: Report encoding/write failures truthfully and keep logs outside machine result streams (#59).
- **FR-011**: Align software, documentation and schemas, describe compatibility changes, preserve historical release bytes, and verify affected online/offline examples.
- **FR-012**: Qualify extracted packages and shared desktop/runtime contracts on configured platform/backend lanes; preserve hidden noninteractive launch and the CI budget.

### Key Entities

- Command definition: command/aliases, option arity, defaults, limits, examples and input descriptions.
- Response: version, request/workspace/session identity, result, classified error and bounded diagnostics.
- Date observation: raw literal, elected/defaulted zone source, precision, instant and uncertainty.
- Durable failure: stable cause and bounded safe context associated with existing work/attempt authority.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every supported help path succeeds inside/outside workspaces with zero created files, queued work or contacted providers.
- **SC-002**: All nine linked issue acceptance matrices pass, including packaged consumers and shared operation tests.
- **SC-003**: Stored/protocol/CLI decimal integer values agree exactly and explicit-offset results agree across timezone hosts.
- **SC-004**: Classified failed work survives restart with unchanged identity, receipts, retry fencing and terminal state.
- **SC-005**: Required local and hosted checks pass on the final PR head; every review thread is resolved and no third round is requested.

## Assumptions

- Existing local/hosted storage, catalog and graph backends retain full support.
- This slice excludes broader human renderers/progress/startup diagnostics (#56-#58), model context (#61), complete documentation repair/presets (#62/#70), and installer/setup work (#63-#69).
- Existing publication authorization covers push and official PR; final merge and product release remain owner actions.

## Clarifications

### Session 2026-10-11

- Machine responses use one versioned envelope: success on stdout, failure on stderr, retaining useful last/result context on failures. Existing exit codes remain 0/1/2/3/4. Compatibility changes are documented explicitly.
- Help/version have structured machine results; human version remains the exact version string. Unqualified group invocation displays contextual help.
- Scalar duplicate options fail; known-speaker remains repeatable. Preserve flexible ordering and literal termination; support equals forms through the same arity model.
- Successful media admission with unresolved metadata/date remains partial success; typed item failures retain their existing durable failure semantics.
- Keep older generic durable causes readable and add classified cause validation without weakening secret, receipt or ownership validation.
