# S004 research

## Credentials

Decision: catalog-independent shared provisioning plus selected persistent secret provider.
Rationale: PostgreSQL opens before runtime dispatch, so catalog-dependent setup would deadlock configuration.
Alternatives: lazy catalog initialization expands lifecycle unnecessarily; environment-only secrets do not satisfy persistence.

Decision: Windows native wincred, macOS Security.framework with interaction disabled, Linux Secret Service DBus without Unlock/prompt invocation.
Rationale: existing go-keyring Unix helpers can prompt and Darwin shells security. Explicit native failures are preferable to unexpected UI.
References: https://specifications.freedesktop.org/secret-service/latest/ and existing exact module source.

Decision: reviewed Argon2id and AES-GCM, fixed bounded parameters, random salt/nonce, exclusive atomic synced ciphertext writes.
References: https://pkg.go.dev/golang.org/x/crypto/argon2 and https://pkg.go.dev/crypto/cipher#NewGCM.
Alternative: custom cryptography/plaintext rejected by constitution.

## Catalog and recovery

Decision: freeze historical DDL/digests before new fields; additive typed current library/model/workflow state, all exported/restored with authority expired.
Rationale: mutable reflective domain definitions currently change historical migration digests. Insert-only metadata cannot meet removal requirements.
Alternative: selected-current pointers and retained old captures violate #6/#21.

Decision: real fenced workflow executor, distinct from the existing timer qualification job.
Rationale: marking a timer succeeded would not demonstrate import/download completion. Stable subordinate identities reconcile lost responses and retries.

## Extraction and dates

Decision: stable scratch bytes, ExifTool qualified duplicate groups/structured JSON with inherited configuration disabled, ffprobe format/streams/chapters JSON, separate bounded stderr.
References: https://exiftool.org/exiftool_pod2.html and https://ffmpeg.org/ffprobe.html.
Rationale: stderr warnings must not corrupt JSON; decimal/source clocks use exact integers, original duration is independent of subtitles.
Alternative: original-path extraction followed by a different copy permits mismatched provenance.

Decision: capture one batch zone/rules, preserve literals/conflicts and resolve explicit folds/gaps only.
Rationale: owner-approved date uncertainty is scoped to dates and never rejects otherwise usable media.

## Scope

Complete #5/#6. Apply #21 current metadata/timing contract without implementing annotated Cue JSON. Inference, full provider presets and package release remain their existing issues. No model-weight download is required just to inspect/import media.
