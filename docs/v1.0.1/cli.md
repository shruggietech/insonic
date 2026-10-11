# CLI invocation and results

Run `insonic --help` for the installed command inventory. Every command and group accepts `-h`, `--help` and `help COMMAND`. A group without an operation displays its commands. Help succeeds before workspace discovery, input-file reading, credential input, provider access, runtime startup or mutation. `--version` is an alias for `version`.

```text
insonic help media import
insonic settings set --help
insonic models speaker list -h
```

## Arguments and JSON files

Global `--workspace DIRECTORY`, `--request-id UUID` and `--json` options can appear before or after command options. Scalar options appear once. Value options accept either `--name VALUE` or `--name=VALUE`; Boolean switches take no value. A value is consumed as a whole token, including a value beginning with `--`. Repeatable `--known-speaker` import selectors and roster `--speaker` selectors retain every occurrence. Contextual help lists each command's accepted options, required operands, enums, ranges, defaults and repeatability.

`--` ends option and help recognition. Following filenames or text remain literal data. For example, `insonic search -- --json` searches for the text `--json`, and `insonic catalog export -- --help` writes the explicitly selected filename `--help`. Help flags that precede the terminator resolve before effects.

`--input FILE` reads a complete JSON file. Contextual help prints an illustrative JSON-file template and its applicable packaged input contract, including optional fields and conditions. Replace illustrative UUIDs, revisions, paths and model selections with current values. Input files do not contain an IPC request envelope. A model register/acquire command instead accepts a model-manifest file as its operand. Structured definitions and model manifests retain their own documented contracts.

```text
insonic --workspace LIBRARY settings show --json
insonic --workspace LIBRARY settings set --input settings.json --json
insonic --workspace LIBRARY models list --speaker SPEAKER_UUID --limit 100 --json
insonic --workspace LIBRARY work wait WORK_UUID --timeout-ms 30000 --json
```

`models base` aliases the base-model commands; `models download` and `models base download` alias acquisition. `models list --speaker UUID` uses the same speaker-model listing as `models speaker list --input FILE`. Omit `--after` for its initial page. `search` constructs a normalized `query run` input; `query native` constructs a native `query run` input using `--dialect`, `--file` and optional `--parameters` files.

Catalog migration/export/restore and portable backup operations are advanced maintenance commands. Mutating maintenance requires exclusive workspace ownership. `runtime serve` is the internal noninteractive owner normally launched by clients; contextual help keeps its syntax available.

## Machine output

Every `--json` result uses the current `runtime-response` envelope. Help and version have structured results; local initialization, configuration and maintenance use the same envelope as live runtime operations. Request IDs correlate responses, and workspace/runtime IDs appear when known. Integers remain JSON integers with their exact decimal value, including signed 64-bit clocks and identifiers above JavaScript's exact Number range. Consumers must use an exact-integer parser for such fields.

| Outcome | Stream | Exit status |
| --- | --- | --- |
| Success | stdout | 0 |
| `invalid_request` | stderr | 2 |
| `incompatible_version` | stderr | 3 |
| `unavailable` | stderr | 4 |
| Other classified failure | stderr | 1 |

A failed wait response retains both its last structured `result` and its top-level `error`. Inspecting failed work through `show` or `results` succeeds as an inspection operation; the work's own state and error remain in the result. A wait timeout reports `unavailable` and leaves durable work running. Encoding or writing output must succeed before a command reports success.

## Dates, credentials and durable failures

An entered recording timestamp containing `Z` or a numeric offset determines its instant when the timezone policy was omitted or `local`. Admission captures the local zone for unqualified wall times and preserves that implicit provenance through replay. An explicitly selected timezone still checks agreement with an entered offset. Date-only precision, unresolved folds/gaps and original literals remain inspectable; no midnight or offset is invented for an unknown time.

Vault add/replace/delete/unlock/load commands use bounded JSON on protected stdin. Deletion requires a passphrase consistently with a stopped, locked or unlocked runtime. Native/session deletion needs no stdin. See [credential commands](secrets.md) for protected input and credential-value forms.

Durable work retains generated classified causes across reload, result paging and snapshot restore. Conflicts, unavailable dependencies, checksum mismatches, input limits, acquisition deadlines and cancellation remain distinguishable. Per-item errors remain in item results; heterogeneous failed causes use `mixed_item_failure`. Historical generic `operation_failed` records keep their original meaning and receipts. No raw provider body or child output becomes a durable error message.

The current reader supports persisted insonic 1.0.0 documents alongside the current document version. Existing workspace/profile identities, revisions, accepted document/model bytes and hashes remain intact. New transport envelopes and package identities use the current release version. A new portable backup uses the current envelope while retaining its historical source profiles and proofs. Older product versions are not promised to understand newly classified causes or current response shapes.

Importing a supported historical tools file with `media tools FILE` or `processing tools FILE` creates a current-version setting while preserving the source file and executable integrity pins. Direct `settings set` payloads use the current input contract.

A supported historical media-import manifest is normalized in the request copy after admission; the source file remains unchanged. Model register/acquire retains an immutable historical manifest's declared version and content digest, using its packaged historical schema inside the current request envelope.
