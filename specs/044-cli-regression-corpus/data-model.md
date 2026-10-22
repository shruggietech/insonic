# Data model

- Corpus version 1: unique bounded fixture/scenario IDs; each fixture references confined repository sources/generators, provenance/license and bounded byte/semantic expectations. Scenarios name existing assertion owners and valid fixture IDs. Unknown members/references, escaping paths and duplicate IDs fail validation.
- Installed authority: command paths and accepted global/command options with value/required/repeatable metadata derived from contextual help. No second maintained command list.
- Observation: generated command/option/scenario/assertion IDs, counts and outcome class; no argument values or raw output. Help-only and parser-only observations remain separate from operation assertions. Failure groups retain schema-owned code/reason values; human-only failures without an envelope remain unclassified. Invocation-only and unverified rows remain separate from asserted success/expected failure.
- Ledger: versioned corpus identity, executable SHA-256, platform/product version, optional verified package archive/inventory source proof, scenario dispositions and complete dynamic command/option rows. Bounded output; source-only companion gaps remain explicit.

Validation bounds: inventory at most 64 KiB, at most 100 fixtures and 32 scenarios; IDs match `[a-z][a-z0-9-]{0,63}`, expectation/provenance strings at most 512 characters. Aggregate at most 20,000 non-help observations; serialized ledger at most 1 MiB. Exceeding a bound fails qualification rather than silently truncating coverage.

- Windows launcher receipt version 1: pinned upstream SHA-256, exact transformed unsigned SHA-256 and original/UTF-8 manifest hashes per language. Corresponding sources retain the original .bin and two portable Python recipe files, all included in the source inventory. Independent verification reproduces exact unsigned bytes; an explicitly verified signing receipt permits only checksum/certificate-envelope normalization. Existing unsigned verification remains exact.
