# Research and decisions

## 2026-10-19

- Decision: Explicit registry guidance/example metadata. Rationale: commands.go derives examples from display operands and uses input.json for enum options. Alternatives: duplicated prose or a new parser framework.
- Decision: Execute emitted examples in the existing installed harness with a coverage ledger. Rationale: #119 requires semantic execution. Update root-help consumers and aliases together.
- Decision: Structured fixed validation causes and known constraint/location messages. Rationale: the schema validator retains details discarded by main.go. Never serialize raw validation errors, which can echo values or unknown property names.
- Decision: Preserve 1..600000 durations through schema/catalog forward migration. Rationale: parser/help promise ten minutes; request, catalog and DDL cap at one minute. Freeze historical schema-9 authority and retain old snapshot restoration.
- Decision: Unicode code points for public character limits; separate byte budgets unchanged. No normalization.
- Decision: Preserve exact model identities and quote/escape them in human output. Rationale: manifest validation is reused for retained installs/snapshots and identities hash original bytes. Global tightening can strand history.
- Decision: Suppress only unestablished failed results before zero work/publication converters manufacture maps. Never delete valid false/zero fields globally.
- Decision: Reserve summaries for identity, every task conclusion/correction, continuation and verbosity before optional details. Raising limits would leave ordering defective.
- Decision: Display valid UTF-8 interior and label incomplete byte boundaries in capture pages. Preserve exact runtime offsets/base64 and authority.

Three read-only Spec Kit research agents examined help/qualification, renderer/receipt and validator/catalog seams. Recommendations were compared with approved scope before adoption.
