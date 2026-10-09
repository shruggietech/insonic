# Model references and acquisition

## Exact identities and short references

Downloaded base models declare exact names, model versions, upstream revisions,
capabilities, license metadata and every required file's role, URL, SHA-256 and
size. Name and version identify one immutable installation. A conflicting
manifest for that pair is rejected; changed files require another version.
Registration does not imply its bytes are available.

Model references accept an installation UUID, `base:UUID`, `speaker:VERSION_UUID`,
a workspace alias, or `source:NAME/SELECTOR`. Aliases use lowercase ASCII names
of 1-64 characters, beginning with a letter and containing letters, digits,
periods, underscores or hyphens. They cannot collide with UUID syntax. Alias
targets declare one operation and exact base, speaker-version or hosted target.
Names are unique within their workspace. Ambiguous or unknown references never
select the first returned version or guess a download URL.

Alias records have stable UUIDs and optimistic revisions. Retarget and remove
require the current expected revision. Removal preserves a tombstone so an old
edit cannot change a recreated alias; recreation uses the existing ID/revision.
Changing an alias affects future submissions, never accepted job elections.

Direct bundle acquisition reports an exact base target with an empty operation;
accepting the bundle does not establish inference compatibility for a task.
Pipeline inspection retains an unavailable reference as a `missing` selection
with an `unresolved` target and the intended operation. It reports no model ID,
manifest digest or upstream revision until resolution succeeds.
An alias declared for another operation retains its exact target and reports
the operation mismatch as incompatible.

```sh
insonic models resolve speech-main --operation transcription --json
insonic models resolve base:MODEL_ID --operation transcription --json
insonic models alias list --json
insonic models alias show ALIAS_ID --json
insonic models alias set ALIAS_ID --input alias-update.json --json
insonic models alias remove ALIAS_ID --expected-revision REVISION --json
```

For example, `alias-update.json` creates a transcription alias to a registered
installation. Replace the example UUIDs with the selected stable IDs; retargeting
uses the returned positive revision in `expected_revision`.

```json
{
  "expected_revision": 0,
  "alias": {
    "id": "10000000-0000-4000-8000-000000000051",
    "name": "speech-main",
    "state": "active",
    "target": {
      "kind": "base",
      "id": "10000000-0000-4000-8000-000000000052",
      "operation": "transcription"
    }
  }
}
```

## Configured discovery

Sources are explicit revisioned workspace records with a unique short name,
catalog URL, optional opaque credential UUID and an explicit loopback HTTP
exception. Discovery reads bounded metadata and reports declared entries,
complete manifests, compatibility and availability separately. A discovered
entry is not presented as a verified installed model. Custom manifests remain
supported; no mandatory curated-model list is imposed.

```sh
insonic models source list --json
insonic models source set SOURCE_ID --input source-update.json --json
insonic models source show SOURCE_ID --json
insonic models discover SOURCE_ID --json
insonic models resolve source:custom/speech:v1 --operation transcription --json
insonic models source remove SOURCE_ID --expected-revision REVISION --json
```

A source update has `expected_revision` and a `source` object containing `id`,
`name`, `state`, `url`, optional `credential_id` and `local_http`. New records use
expected revision zero; updates/removal use the returned revision. Source URLs
and file URLs must follow the configured transport policy: HTTPS by default,
explicit loopback HTTP where permitted, no password/user information or query
credentials, and same-origin redirects only. Saved credential values are not
included in source records, manifests, durable work or status.
Catalog content cannot authorize loopback file downloads. Such files require
an owner-configured loopback source with its local opt-in enabled. An explicit
owner-supplied manifest retains its own local transport option.

The implemented source protocol is a `model-catalog` document with
`schema_version`, and up to 128 `entries`, each containing `selector` and a
complete `manifest`. Responses are bounded to 4 MiB. Selectors may represent
tags or branches, but resolution freezes the returned exact bundle and its
declared upstream revision. Catalog providers should supply immutable upstream
revisions wherever available; unknown declarations remain explicit. Every file,
including tokenizer/configuration auxiliaries, still requires SHA-256 and size.
Updating the catalog later cannot change an existing election.

Direct Hugging Face APIs/CDN downloads are not an implemented source adapter.
An explicitly configured catalog can describe supported URLs; unsupported
provider redirects or formats are diagnosed instead of silently changing the
transport policy. [Revision-pinned acquisition guidance](https://huggingface.co/docs/huggingface_hub/main/guides/download)
explains why mutable branches are convenience selectors rather than job identity.

## Compatibility before execution

Optional manifest `compatibility` declares `adapter`, `contract_version`,
`architecture`, `runtime_format`, and optional `sample_rates` and `channels`.
Inspection validates task, required file roles, adapter contract and audio
requirements without loading a model. Available and compatible are different
properties: custom bundles may be registered/acquired while lacking a supported
execution adapter.

Installed inventory includes declared capabilities and compatibility with the
default adapter for each operation. Shared selectors use these summaries to
label incompatible bundles while retaining exact identity and availability.
Reference inspection checks the requested operation; a saved pipeline's
inspection also checks its explicitly selected stage adapter.

The current local adapters require contract version `1`. `faster-whisper`
consumes Whisper/CTranslate2 files named `config.json`, `model.bin`,
`tokenizer.json` and `vocabulary.txt`. `pyannote` consumes Pyannote/Torch files
named `config.yaml`, `embedding/pytorch_model.bin`,
`segmentation/pytorch_model.bin`, `plda/plda.npz` and
`plda/xvec_transform.npz`. Matching is case-sensitive and role paths are confined.
Their mapped engine input is 16000 Hz mono. Older accepted manifests without
explicit compatibility retain their original digests and are checked against
these established role contracts; they are not silently rewritten.

Unsupported task/runtime/architecture/version, missing roles and incompatible
audio requirements are diagnosed before inference. Downloading arbitrary
weights does not make them executable. A failed local selection does not choose
another model, device or hosted provider.

## Acquisition and dependent work

```sh
insonic models register model.json --json
insonic models acquire model.json --json
insonic work show WORK_ID --json
insonic work wait WORK_ID --timeout-ms 30000 --json
insonic models list --json
insonic models show MODEL_ID --json
insonic models verify MODEL_ID --json
insonic models materialize MODEL_ID --json
```

Processing/import elections resolve references once, validate compatibility and
freeze exact IDs, complete manifest digests, upstream/adapter identities and
effective settings. Missing selected bundles acquire automatically before
dependent processing. Saved pipeline inspection remains free of acquisition
and inference. Direct processing fields and import's `--diarization-model-id`
use the same reference grammar; see [pipelines](pipelines.md) and
[import](ingestion.md).

Acquisition returns durable work immediately. Underlying acquisition is shared
by exact workspace/bundle identity; explicit requests and processing consumers
retain independently cancellable parent work. Waiting consumers leave worker
slots free for acquisition. Work inspection identifies frozen selections,
acquisition IDs and phase. A failed/cancelled dependency produces a diagnosed
acquisition failure, not a completed pipeline. Retrying the parent repairs its
original dependency without following a later alias or source change.

Downloads resume bounded partial transfers, verify all hashes/sizes and publish
files before accepting availability. Already available bytes are verified,
not trusted by cache filename. Missing/corrupt bytes require exact recovery.
Cancelled/stale work cannot publish a late processing result. Cancelling one
consumer does not cancel shared acquisition still used by another consumer.

Materialization returns verified local paths with renewable artifact leases.
Release leases after the consumer finishes. Temporary/base caches are separate
from durable speaker profiles and trained artifacts; their cleanup cannot purge
trained output or active readers.

## Portability and speaker versions

Artifacts use configured filesystem or generic S3 storage. Exact manifests,
aliases, sources and model relationships use portable SQLite/PostgreSQL catalog
records with schema migration and snapshot proof validation. Exports contain
opaque credential references, not secret values or transient materialization
paths. Graph relationships remain projections of catalog authority.

`speaker:VERSION_UUID` inspection preserves the existing family, producing
speaker, training run, dataset and artifact lineage. Hosted-only targets retain
their explicitly configured provider/remote handle and report that local
downloadable weights are unavailable. Neither an alias nor a retained artifact
fabricates a valid trained output or a runnable unsupported adapter.
Local processing/import model selections require compatible base bundles.
Hosted aliases support identity inspection; executing a hosted route uses the
existing saved pipeline's explicit endpoint, remote model and adapter settings.
Trained speaker references expose lineage until a compatible consumer is
implemented. Inspection does not make these targets selectable by local adapters.
[Speaker-associated training](voice-models.md) and acoustic identity matching
remain separate delivery contracts.

CLI/runtime and desktop share these operations. Models settings support
reference configuration/discovery; shared selectors and Jobs distinguish exact
identity, availability, compatibility and acquisition. See [desktop](desktop.md),
the rendered [JSON contracts](contracts.md) and [credentials](secrets.md).
