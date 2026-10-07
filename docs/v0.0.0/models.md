# Downloaded models

Downloaded base models have exact names, model versions, upstream revisions,
capabilities, license declarations and ordered file roles. Every file declares
its URL, SHA-256 and byte size. Credentials use UUID references and remain
outside manifests. URLs contain no passwords, query credentials or fragments.
HTTPS is the default; a manifest can explicitly permit loopback HTTP for an
isolated local source. Acquisition follows same-origin redirects only.

Registration stores an exact manifest without claiming its bytes exist.
Name and model version identify one logical installation. Once registered,
a changed manifest for that same pair is rejected; use a new model version for
changed files or provenance. Replaying the unchanged manifest retains its ID.
Acquisition downloads the required files, resumes bounded interrupted partial
downloads, checks their declared sizes and hashes, and publishes verified
artifacts before accepting an available model. A failed or cancelled acquisition
cannot publish an available model record. Recovered work verifies already
published files before accepting completion.

```sh
insonic models register model.json --json
insonic models acquire model.json --json
insonic work show WORK_ID --json
insonic models list --json
insonic models show MODEL_ID --json
insonic models verify MODEL_ID --json
insonic models materialize MODEL_ID --json
```

Acquisition returns a durable work ID immediately. Poll its state and per-file
progress; `work cancel` and `work retry` use that ID. Materialization returns
verified local paths with renewable artifact leases. Release those leases using
the artifact lease operations after the consumer finishes. Verification checks
the bytes in the selected artifact store rather than trusting a cached filename.

Model artifacts use the configured filesystem or generic S3 store. Their
manifests and availability are portable SQLite/PostgreSQL catalog records.
Snapshots omit credential values. This registry is independent of
[speaker-associated training models](voice-models.md); it does not fabricate
speaker identities, training datasets, or inference results. Provider inference
and trained speaker-model creation remain separate implementation work.

See the rendered [JSON contracts](contracts.md) for the base-model manifest
shape and [credential commands](secrets.md) for provider bootstrap.
