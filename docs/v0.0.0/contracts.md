# JSON contracts

## One versioned system schema

insonic uses [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12) for configuration, import, artifact/metadata interchange, portable catalog snapshots, durable job events, graph queries and speaker-training/model manifests. A system-wide master schema contains a registry of all insonic-owned contracts and selects documents through their `kind`. Shared definitions provide consistent identifiers, digests, revisions and artifact references. Cueson retains its own immutable upstream subtitle schema; the master recording embeds its validated current document without redefining that format. Its UUID assignments appear only there; external speaker mappings and other contracts carry current document/cue references, without another assignment list. Local packaged upstream schema bytes provide offline validation.

The master and child schemas live in `schemas/v<version>/`. Every release has the same software, documentation and master-schema version. A document declares `schema_version` and `kind`; the registry resolves its exact child contract. Tagged canonical schema IDs identify release resources, and the packaged registry resolves them offline. Schema files carry descriptions and examples for fields and complete documents. The documentation build reads those fields to produce the contract reference below and packages the JSON files beside it.

## Compatibility and evolution

Model selection separates revisioned convenience aliases and configured source catalogs from immutable elected identities. Runtime requests expose alias/source CRUD, discovery and resolution; processing/import payloads freeze selected bundle manifests and acquisition dependencies. Optional compatibility metadata augments base manifests without rewriting accepted older digests. Portable catalog records retain alias/source revisions and trained/hosted target distinctions. Structural schema validation complements runtime checks for target ownership, complete bundle compatibility and job fences. See [models](models.md) for executable commands and lifecycle.

Recording replacement is a targeted media election with explicit transcript keep, clear or replace and declared-roster retain or clear policies. Keep requires an applicability assertion and validates known source-clock bounds and selected track/channel compatibility. A retained older managed subtitle can become the validated embedded current document during that same acceptance. Failure preserves previous accepted state. Roster read and edit requests carry independent optimistic revision authority; intentional membership remains distinct from document attribution and training evidence. The runtime request and [import rules](ingestion.md) expose these elections consistently.

Preserve backward and forward compatibility where practical. Prefer additive optional fields, stable meanings, explicit extension containers and migrations over silently changing existing values. Readers validate against the document's declared version and negotiate adapter capabilities. An unknown version is reported clearly; it is not silently interpreted with whichever schema is newest. Keep supported older registries available for imports and migrations. Unsupported features can remain preserved as opaque extension values where the contract permits them.

Compatibility is a priority, not an absolute prohibition on useful change. When a change breaks a consumer or persisted contract, describe the affected fields, supported source versions and migration/recovery path in the changelog and the brief release highlights. Release notes retain the final changelog link. A schema revision alone does not prove compatibility; validation and representative import/export/migration fixtures establish the supported combinations.

## Validation and documentation

Speaker-work requests distinguish dataset creation/inspection, elected training, exact speaker-version retrieval, profile CAS updates and matching-only work. Public matching input cannot supply server-frozen recording/model digests or replace the declared roster with a cluster-count hint. Zero score thresholds and ambiguity margins are explicit values. Catalog schema 9 carries portable output/profile/checkpoint authority and origin-qualified mappings; model/dataset manifests validate against their exact child contracts.

The repository checks master registration, matching release identities, schema validity, field descriptions and embedded examples. Every document example must validate both against its child schema and the master. Contract checks also reject malformed identities, contradictory inputs and mismatched kinds/versions. Structural validation is complemented by application checks for source ownership, time bounds, permissions, revision ordering and backend capabilities; JSON Schema cannot prove those relationships by itself.

The generated reference is read-only output. Edit descriptions, examples and field constraints in the authoritative JSON files, then rebuild documentation. The same packaged definitions serve CLI validation, adapter interchange and offline help. [Catalog relationships](schema.md), [import behavior](ingestion.md) and [speaker model lineage](voice-models.md) explain the domain rules around these payloads.

<!-- schema-reference -->
