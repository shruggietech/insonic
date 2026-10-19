# Feature Specification: CLI contract and operator-feedback correctness

**Feature Branch**: `codex/039-cli-contract-correctness`

**Created**: 2026-10-19

**Status**: Implemented review candidate; owner-approved residual #154 investigation deferred

**Input**: Run S039 end to end under autopilot, closing #116, #117, #118, #119, #120, #121, #131, #132, #133, #134 and #144, with proven native fixes contributing to still-open #154. Push and publish the PR automatically, address every review finding with at most two review rounds, verify the exact final head, and stop before owner review and merge.

## User Scenarios & Testing

### User Story 1 - Choose and execute a valid command (Priority: P1)

An operator uses root or contextual help to choose a task, understand its inputs and execute the example using current library identities and documented fixture replacements.

**Why this priority**: Help must be an executable account of the interface rather than a list of unexplained tokens.

**Independent Test**: Enumerate every accepted command, option and operand, inspect its guidance, and execute each generated example in isolated libraries using only documented substitutions.

**Acceptance Scenarios**:

1. **Given** any command/group, **When** help is requested, **Then** its purpose and accepted operands/options explain units, defaults, required or conditional status, repeatability, exclusions, path rules and applicable lifecycle without starting work.
2. **Given** enum, maintenance, paging, protected-input or positional-document commands, **When** the documented workflow runs, **Then** the examples use actual accepted values, identities, cursor forms and templates, including conditional prerequisites and meaningful follow-ups.
3. **Given** a synthetic job duration at 1, 60000, 60001 or 600000 milliseconds, **When** it is submitted, **Then** it is accepted consistently and can immediately be cancelled; zero and 600001 fail with the supported interval.

### User Story 2 - Understand and correct a rejected request (Priority: P1)

An operator receives a stable cause and a command-specific correction for malformed input, unsupported shape or version, violated constraints, absent local prerequisites and output destination failures.

**Why this priority**: Correct rejection without a useful explanation leaves the operator unable to proceed.

**Independent Test**: Run the maintained malformed-input, protected-input, Unicode and destination matrix through the installed CLI and compare classified errors, streams and exits across every diagnostic mode.

**Acceptance Scenarios**:

1. **Given** empty, malformed, duplicate-key, extra-document, null, array, unknown-field, enum/type/range or unsupported-version input, **When** it is rejected, **Then** generated diagnostics identify the failed shape or constraint and a known field location or parse position when available without repeating input values.
2. **Given** missing workspaces, an existing recording-export destination, or a missing/unwritable catalog-export parent, **When** the command fails, **Then** its correction describes the actual prerequisite or output-path condition and preserves the established exit meaning.
3. **Given** ASCII, composed/decomposed accents, CJK or emoji at a stated character limit, **When** the value is validated, **Then** the documented Unicode character count agrees across public contracts and runtime validation; protected credential byte ceilings remain explicitly byte based.
4. **Given** blank or multiline model identity metadata, **When** it is registered or displayed, **Then** the documented preservation policy and escaped human representation make its exact identity unambiguous, without trimming existing identities.

### User Story 3 - Trust observed results and bounded summaries (Priority: P1)

An operator can distinguish failed inspection or bootstrap from observed state and can find current identities, revisions, digests, task conclusions and supported next commands before optional detail consumes the display budget.

**Why this priority**: Invented success or omitted task conclusions can lead to the wrong next operation.

**Independent Test**: Exercise warm/cold credential bootstrap, local value-result failures, rich transcripts, packaged prerequisite inventories and large pages in human and machine modes.

**Acceptance Scenarios**:

1. **Given** failed vault/session bootstrap or runtime startup, **When** credentials unlock/load returns, **Then** it retains its classified failure without claiming unlocked/loaded; successful warm/cold invocations still return the observed success.
2. **Given** a failed location/artifact/other value-struct handler with no established result, **When** output is emitted, **Then** no invented zero receipt, empty location or false existing-workspace observation appears; legitimate observed partial and false/zero values remain visible.
3. **Given** a tiny rich transcript, many prerequisites or a large search page, **When** human output reaches its bound, **Then** identities/revisions/digests, every task conclusion/correction, paging authority and verbosity context remain visible before summarized optional detail.
4. **Given** metadata, facts or dates capture pages, **When** displayed in human mode, **Then** readable UTF-8 text and exact valid continuation guidance appear, including a split within a multibyte sequence; machine output preserves exact base64 bytes and authority.

## Edge Cases

- Help before workspace discovery, file reading, protected stdin or mutation; alias coverage and `--` literal operands.
- Model/provider-selected strings that resemble terminal controls; whitespace-only names, CR/LF, Unicode and legacy immutable identity values.
- Exact signed 64-bit integers, narrow output budgets, mixed readiness conclusions, absent paging routes and cursors on result-only pages.
- Wrong passphrase, rejected session bootstrap, startup failure, failed artifact publication, legitimate partial observations and valid false/zero fields.
- File exists, missing parent, access denied, directory operand, write/sync/close failure and simulated exhausted space; never overwrite a preexisting destination.
- Protected-input byte boundaries; invalid JSON locations without echoing private keys/values, arbitrary child/provider output or document content.
- UTF-8 byte pages split inside a character, empty/complete pages and stale capture/document authority.

## Requirements

### Functional Requirements

- **FR-001**: Every accepted command/group MUST have a concise task description and a complete reviewed option/operand inventory with meaning, units, defaults, required/conditional status, repeatability and relevant exclusions (#118).
- **FR-002**: Help MUST explain absolute/relative paths, pre-workspace operations, source/recording identities, immutable retry inputs, protected stdin and durable acceptance versus completion where applicable (#118).
- **FR-003**: Generated examples and templates MUST use executable enum values, correct cursor types/names, positive deletion revisions, operation-specific backup operands/ownership and valid training definitions. Unusable options MUST be removed or made meaningfully supported (#119).
- **FR-004**: Every generated example MUST execute with only documented identity, revision, path and fixture selections substituted. Matching strings or schema-only validation MUST NOT count as semantic success (#118, #119; contribution to #114).
- **FR-005**: Rejections MUST distinguish JSON syntax, duplicate/extra keys, shape/type/enum/range, unsupported version, stale revision and local prerequisites using stable generated causes and bounded useful constraints/locations (#117).
- **FR-006**: Missing workspace and existing recording-export paths MUST have operation-specific recovery; catalog export MUST distinguish existing destination, missing parent, access denial, directory operand and exhausted-space/write failures (#117, #131).
- **FR-007**: Parser, help, public request validation and execution MUST support job durations from 1 through 600000 milliseconds and reject values outside that interval consistently (#132).
- **FR-008**: Public character limits and runtime text validation MUST count Unicode code points consistently, without normalizing stored identities; byte limits MUST be labelled and independently enforced (#133).
- **FR-009**: Model identity text MUST preserve its exact registered content, including historical values; intentional blank/multiline preservation MUST be documented and displayed with unambiguous escaping (#134).
- **FR-010**: Credential unlock/load MUST report success state only after successful bootstrap, preserving failure exit/cause and protected-input confidentiality (#116).
- **FR-011**: Failed handlers MUST omit unestablished results while preserving accurate explicitly observed partial results and valid false/zero fields (#120).
- **FR-012**: Bounded human output MUST lead with current identities, revisions, digests, state, all task conclusions/corrections and applicable continuation. It MUST reserve warning/verbosity space, summarize embedded documents/inventories and retain exact numeric tokens (#121).
- **FR-013**: Omitted detail MUST identify an actual valid next command. Commands without paging MUST offer their exact JSON retrieval command rather than an invented page route (#121).
- **FR-014**: Human media capture metadata/facts/dates MUST display readable text with authority and continuation while machine pages retain exact base64 bytes, including multibyte boundary cases (#144).
- **FR-015**: Human, quiet, verbose, debug and JSON modes MUST preserve exit meanings and stream selection. Generated diagnostics MUST never disclose protected stdin, credentials, private request contents or arbitrary provider/child output.
- **FR-016**: Qualification MUST extend existing installed-binary tools with versioned deterministic fixtures and explicit command/option coverage, using isolated owned profiles/libraries and hidden noninteractive Windows processes with observed cleanup (#114 contribution).
- **FR-017**: Shared runtime and public schemas MUST remain aligned with CLI and GUI consumers. Existing immutable identities, accepted data and historical release records MUST remain intact.
- **FR-018**: Required repository, affected site/product, installed-package and exact-head hosted checks MUST pass within the 660-second CI budget. Every review finding MUST be addressed/resolved, with at most two review rounds, before unmerged owner handoff.

### Key Entities

- **Command guidance**: Accepted command path, task purpose, operand/option contracts, example inputs, prerequisite workflow and valid follow-up.
- **Generated failure**: Stable category/cause, bounded constraint/location and corrective action, independent of private input values.
- **Observed result**: Established operation facts or an accurately labelled partial observation; absent when no result was established.
- **Human summary**: Essential identities, conclusions and continuation followed by bounded optional detail.
- **Qualification scenario**: Deterministic fixture authority, documented substitutions, expected state/stream/exit and owned teardown.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every accepted command/group/option/operand has reviewed guidance and every generated example is recorded as an executed controlled scenario with an expected semantic outcome.
- **SC-002**: All malformed-input, Unicode, credential-bootstrap and filesystem boundary fixtures return the expected stable cause, correction, stream and exit without private-input disclosure.
- **SC-003**: Every reported eleven-issue reproduction has an installed-binary regression and meets its complete acceptance criteria; #114 and #147 remain open for their broader coverage.
- **SC-004**: Essential identities, task conclusions and valid next commands remain visible within existing human output bounds; complete machine results and exact integers remain available.
- **SC-005**: All elected checks pass on the final PR head within 660 seconds, all review findings are resolved after at most two rounds, and the PR remains unmerged for owner review.

## Assumptions

- Baseline is merged main `1ab36def9dc9cb700e9dcb192c3ff49cbc4c35e5`; S039 uses its own matching branch/directory and eleven-issue mapping.
- #115, #126 and #135-#139 remain separate defects. Owner approval includes the proven #154 native lifecycle/input/reference fixes; the unexplained startup failures remain in open #154 outside S039 completion. Their required existing checks still apply; no known failure is converted into a successful receipt.
- This slice contributes targeted maintained fixtures to #114 and updates #147 evidence without claiming either broad tracker complete.
- This is a source correction, not a new product release, installation into an owner library, or website publication.

## Clarifications

### Session 2026-10-19

- Q: Which job bound is authoritative? A: Preserve the advertised ten-minute capability, 1..600000 milliseconds, across parser, schema and execution; cancel accepted synthetic boundary jobs immediately.
- Q: What does a character mean? A: A Unicode code point, matching the public schema; do not normalize stored strings or change protected-input byte budgets.
- Q: Should existing model identities be trimmed or rejected on read? A: No. Preserve exact identities and document intentional preservation; quote/escape whitespace and control characters in human identity displays.
- Q: What is the publication boundary? A: Push/open the official PR automatically and handle at most two review rounds; stop with green exact-head CI and resolved reviews before owner merge.
- Q: Does partial fixture coverage close #114/#147? A: No. Preserve complete per-defect acceptance and record the broader trackers as still open.

## Owner-approved scope extension (2026-10-19)

The owner explicitly selected "Include #154 repair in S039." The twelve-issue slice now includes the native startup/lifecycle repair deferred by S038. Push/PR authorization and the exhausted two-round external review cap persist. Stop before owner merge; post-merge main verification remains unavailable until that merge occurs and must be reported explicitly.

### User Story 4 - Correct demonstrated native Windows ownership and input defects (Priority: P1)

An operator and the disposable qualification runner can create the native WebView, reach DOM readiness, activate the actual File/Exit, window-close and Alt+F4 routes, and observe normal host/browser cleanup without disrupting the independently owned runtime or durable work.

**Acceptance scenarios**:

1. Given retained controller-busy and startup-watchdog evidence, when the delivered corrections are reviewed, then demonstrated ownership/input defects and their regression proofs are distinguished from the unexplained startup failures retained in #154. No additional diagnostics are part of closeout.
2. Given the demonstrated failing lifecycle, when its correction runs, then a regression reproduces the old failure or invariant violation before the fix and verifies the corrected boundary; success cannot be manufactured by skipped native actions, forced-kill receipts or larger timeouts.
3. Given cold/warm native launches with and without a workspace, when actual menu, close and Alt+F4 qualification completes, then owned playback is released, held durable work and runtime session survive, and owned WebView descendants have exited before profile reuse/removal.
4. Given every ZIP/MSI/setup/recovery proof, when final qualification joins them, then the original exact-source/byte and failed-producer gates still apply and complete exact-head CI remains within 660 seconds.

### Additional functional requirements

- **FR-019**: Native investigation MUST retain bounded action/stage/HRESULT and owned browser/profile lifecycle evidence without exposing protected inputs or modifying ambient owner profiles/processes.
- **FR-020**: Demonstrated owned-process, controller/COM-reference and native-input invariant violations MUST be corrected proportionally, with failing-before/passing-after regressions explaining each corrected boundary. The unexplained pre-DOM and two-second UI-join startup signatures remain open in #154 and outside S039 completion by owner direction. No private WebView cause, profile contention, resource-pressure cause or link between distinct signatures may be claimed without evidence.
- **FR-021**: Real Windows cold/warm menu, close and Alt+F4 checks MUST retain workspace/playback/durable-runtime assertions and observe owned browser release before fixture profile reuse/removal. No skipped action, forced-kill success, longer watchdog or weaker proof may confer acceptance.
- **FR-022**: The eleven closure mappings and proven-fix contribution to still-open #154, affected source/product/site checks, exact-head CI within 660 seconds, resolved existing reviews and the two-round cap MUST be preserved before the unmerged owner handoff. No further diagnostics are authorized. Post-merge main remains an explicit later observation.

### Additional success criterion

- **SC-006**: The demonstrated native ownership/input/reference defects have focused causal regressions and passing actual Windows startup/Exit and delivery acceptance. The remaining unexplained startup failure is explicitly retained in open #154, without claiming its cause repaired. A green PR does not claim post-merge-main or owner-installed behavior.

### Scope clarification

- Q: Should the native cause repair remain separate after repeated final-head failures? A: No. The owner explicitly includes #154 repair in S039. The two-review cap is unchanged.

## Owner-approved closeout (2026-10-19, after the bounded diagnostic)

The owner directed: "Keep #154 open for the unexplained startup failure and finish S039 with the proven fixes. No more diagnostics. Get PR #156 ready for my final review and merge ritual, but leave it unmerged." This latest direction refines the earlier native repair extension. S039 delivers the original eleven issue closures and the demonstrated native process/controller/COM, menu-focus/system-key and resource-deferral corrections. The unexplained startup signatures and their cause repair stay in #154; T053 is deferred outside this slice, not reported complete. Existing native/package gates remain mandatory. Push/PR authorization persists, both external review rounds are already used, and merge remains with the owner.
