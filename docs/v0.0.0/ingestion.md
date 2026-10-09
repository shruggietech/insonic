# Import, metadata and dates

## A definite import workflow

Import accepts local audio/video, a batch manifest or an acquisition adapter's output. New admission stores canonical audio in the configured artifact store. Existing copied/reference media remains readable; new reference admission reports an explicit unsupported request. A saved import preset supplies the destination, timezone policy, date precedence and optional processing pipeline. A configured batch runs without approving every file or stage.

Capture metadata from the original bytes as soon as a stable readable source is available, before transcoding, resampling, remuxing, stripping tags or producing a proxy. For remote sources, retain available acquisition metadata before download and extract embedded metadata from the downloaded original before transformation. A source that has already lost metadata cannot have those missing fields reconstructed as facts.

1. Accept local files, a batch manifest or an acquired original.
2. Read stable original bytes and compute their identity.
3. Capture embedded metadata and stream facts.
4. Select and validate an explicit or embedded transcript, then prepare canonical audio.
5. Publish verified candidates and atomically commit the media, optional current document, observations and selected-date state.
6. Make the accepted library entry available and reconcile unused managed candidates.
7. Return durable work and per-item admission results. Processing presets remain recorded inputs until downstream processing is implemented; no unexecuted processing job is reported as complete.

The admission receipt binds original and canonical digests, byte sizes, track-clock maps, acquisition/import times, metadata report digests, extractor versions/options, probe results, effective date policy and selected artifact locations. Detect a source changing during copy or extraction and retry from stable bytes instead of joining a report to the wrong asset. Derived processing starts only after canonical admission and the capture attempt are recorded.

## Metadata preservation

EXIF is one supported metadata family; audio/video can also carry XMP, ID3, RIFF/BWF, QuickTime and other container tags. Capture all available families, container-level tags and per-stream facts, rather than requiring every file to contain EXIF. [ExifTool](https://github.com/exiftool/exiftool/blob/master/README) is the default embedded-metadata extractor; [ffprobe](https://ffmpeg.org/ffprobe.html) supplies complementary container, stream and codec facts. Release packages pin these tools and include their licensing notices.

Persist the current raw extractor report and separate typed normalized observations. A refresh replaces the current computed metadata and physically retires superseded reports after active materialization leases are released. Keep canonical audio and the current document source envelope; preserve owner-owned input files. Preserve tag group/path, duplicate instance, original value, units, precision and warnings. ExifTool JSON can suppress identical tag names unless instance-qualified groups are requested; the extraction contract must retain duplicates and structured values. [Official extraction documentation](https://github.com/exiftool/exiftool/blob/master/html/exiftool_pod.html) describes these options. Unknown fields remain in captured reports when available. A retired original cannot supply fields omitted by an extractor; bounded capture reports its coverage rather than reconstructing missing facts. Report truncation and coverage explicitly.

For a readable legacy local reference, retain the extracted report and available metadata payload artifacts immediately even though media stays outside managed storage. Later source loss leaves those captured observations intact; it cannot guarantee recovery of an unparsed field that existed only in the missing original. State that availability limit, and use canonical managed admission for durable audio.

Capture states are `captured`, `no-embedded-metadata`, `partial`, `unsupported` and `failed`. Only a successful extraction with no embedded fields establishes `no-embedded-metadata`. Unsupported metadata extraction and errors retain the capture attempt alongside otherwise valid canonical audio and allow retry. They do not reject otherwise usable media or claim all metadata was captured. If a limit prevents complete extraction, keep the input item available with a visible partial state.

Store filename extension, acquisition-declared MIME type, byte-detected MIME type, extractor file type and probed container/codec separately. A normalized preferred MIME type records the resolver/version and disagreements. Do not overwrite observations with user edits. Metadata can be inaccurate before arrival; capture preserves what was received rather than certifying its truth. The details/CLI metadata view exposes raw and normalized observations with their source.

## Origination dates and timezones

Origination means the recording's date/time, distinct from publication, retrieval, import, file modification and processing. Import permits `--originated-at` for a timestamp or `--originated-on` for a date without a known time. Use the captured local timezone by default for timestamps without an explicit zone. `--timezone` accepts `local`, `UTC`, an IANA identifier or an explicit numeric offset. Resolve `local` once for the import batch and save the actual zone/offset and timezone-database version; another machine must not reinterpret it later.

The platform timezone bridge resolves Windows/system zone identifiers to maintained IANA rules and retains its mapping/rules version. Use the offset for the entered recording time, not today's offset. An unavailable local-zone mapping reports an unresolved date and accepts an explicit zone; it does not stop media admission.

The default `owner-first` recording-date precedence is an explicit per-item owner value, a valid embedded recording timestamp with its own timezone, an embedded recording timestamp interpreted under the import timezone policy, then unknown. `--date-precedence RULE` selects another saved, versioned rule; the receipt records the effective rule revision. Per-item manifest values override batch defaults. Preserve every competing observation and the reason for selection. The configurable candidate mapping uses recording/creation tags with their documented meanings, never every tag containing a date. Filesystem modification time is available as a separately labelled observation and becomes a recording date only through an explicit policy or owner choice.

The [IANA timezone database](https://www.iana.org/time-zones) supplies historical offset/daylight-saving rules. Store the entered wall-time literal, zone source, resolved offset, UTC instant when resolvable, precision, selection basis and assumption flags. A date-only input retains day precision rather than inventing a recording time. Approximate dates retain bounds; unknown dates stay in the undated timeline group. QuickTime integer date tags are particularly uncertain: [ExifTool's upstream notes](https://github.com/exiftool/exiftool/blob/master/html/TagNames/QuickTime.html) describe cameras storing local values despite the format's UTC convention. Preserve the raw value and recorded interpretation policy rather than silently converting every such tag as UTC.

An unambiguous valid time resolves automatically. An explicit IANA zone and numeric offset must agree if both are supplied. For a repeated daylight-saving wall time, provide an offset or `--dst-fold earlier|later`. A nonexistent daylight-saving wall time needs a corrected value or explicit `--dst-gap shift-forward` policy. No silent DST adjustment is the default. Only the affected date observation remains unresolved; media admission and other batch items continue. Noninteractive mode reports partial completion and the required disambiguation in JSON. Missing metadata or an unknown date never triggers a compulsory review queue.

## CLI and batch contract

The CLI exposes import, inspection and date correction through these operations:

```sh
insonic media import FILE --copy --originated-at 2026-10-04T14:30:00
insonic media import FILE --copy --originated-at 2026-10-04T18:30:00Z
insonic media import FILE --copy --originated-at 2026-10-04T14:30:00 --timezone America/New_York
insonic media import FILE --copy --originated-on 2026-10-04
insonic media import --manifest inputs.json --preset local --json
insonic media raw MEDIA_ID --json
insonic media set-origin MEDIA_ID --revision CURRENT_REVISION --originated-at 2026-10-04T14:30:00 --timezone UTC
```

A JSON manifest carries a version, batch defaults and ordered items. CSV supports the same flat per-item fields. Each media item has a source and optional transcript (`subtitle` is a compatibility alias), origination timestamp/date, timezone and preset override. Sources and transcript paths are resolved relative to the manifest location unless explicitly absolute. Do not put credentials in manifests; use configured credential IDs.

Manifest files and normalized durable import input are bounded to 8 MiB, with
at most 10,000 items. Path resolution and CSV-to-JSON expansion count toward the
normalized input budget; oversized input is rejected before enqueue. The local
request frame allows that input plus its envelope. Responses remain bounded to
1 MiB and use the result/report paging described below.

Example:

```json
{
  "kind": "import-manifest",
  "schema_version": "0.0.0",
  "defaults": { "copy": true, "timezone": "America/New_York", "preset": "local" },
  "items": [
    { "source": "audio/session.wav", "originated_at": "2026-10-04T14:30:00" },
    { "source": "video/session.mp4", "originated_on": "2026-10-03", "subtitle": "video/session.srt" }
  ]
}
```

Per-item output reports media ID, byte identity, metadata capture state, date selection/assumptions and queued job IDs. Repeated admission reuses matching immutable bytes/reports while retaining explicit entry creation and metadata revisions. A date correction creates a new owner observation and selection revision and preserves the current raw report. Calendar controls will consume this selected state when implemented. It does not change original tags, rewrite a transcript or rerun recognition.

Import and refresh return a durable `work_id`. A batch with failed admissions
has failed work state and retains
per-item results; `work retry WORK_ID` reconciles admitted entries and retries
the failed items. An admitted item with partial metadata or unresolved dates
remains usable and does not by itself fail the work. Unused canonical candidates from failed admissions are retired using the current-reference
fence, preserving shared or committed current artifacts.

Inspect completion through
`work show`; large batches expose references through
`work results WORK_ID --after-ordinal 0 --limit 100`. Continue from the returned
`next_ordinal`. `media list`, `models list` and `work list` return summary pages
with `next_id`; pass that UUID with `--after` for the next page. Default and
maximum page size are 100. `media raw` returns current capture-bundle paths with
artifact leases rather than placing large reports in IPC frames. Oversized
normalized metadata also returns these reports with `metadata_inline: false`.
Release each lease after reading using `artifacts lease-release`.

HTTP acquisition has configurable byte and time budgets. The defaults are 64 GiB
and ten minutes per source. Set `--acquisition-max-bytes` and
`--acquisition-timeout-ms`, or the equivalent `acquisition_max_bytes` and
`acquisition_timeout_ms` batch/per-item manifest options, for larger recordings
or slower configured sources. Values must be positive; item options override
batch defaults. Declared and streamed responses exceeding the ceiling fail,
cancelled/timed-out downloads stop, and partial acquisition files are removed.
Local imports do not use this remote-download ceiling.

Credential-bearing acquisition uses HTTPS. Explicit `--local-http` (manifest
`local_http: true`) permits credential-bearing HTTP only for a loopback source;
it cannot permit plaintext remote authentication. Ordinary HTTP sources without
credentials remain supported. Authentication belongs in credential IDs, and
source URLs reject common query-authentication aliases before durable enqueue.

JSON date input can preserve approximate `originated_earliest` and
`originated_latest` bounds as exact `{iso, unix_ns}` UTC instants. Either bound
may be absent; two bounds must be ordered and their text and integer values must
agree. A bounded observation has range precision and no invented exact recording
instant. Per-item bounds replace batch bounds and cannot be combined with an
exact `originated_at` or `originated_on` value.

The CLI accepts `--originated-earliest` and `--originated-latest` RFC3339
timestamps for import, refresh and `set-origin`. It normalizes explicit offsets
to UTC and constructs exact integer nanoseconds without floating-point conversion.
Either flag may be omitted for an open bound. For example:

```sh
insonic media set-origin MEDIA_ID --revision CURRENT_REVISION --originated-earliest 2026-10-01T00:00:00Z --originated-latest 2026-10-07T23:59:59Z
```

Configure exact extractor files before import using
`insonic media tools media-tools.json`. The rendered tools contract declares
absolute executable paths, SHA-256 values, emitted versions, delegated support
files and an explicit pinned interpreter when needed. Configure while the runtime
is stopped. Missing or mismatched tools yield a visible partial/unsupported
capture and preserve usable media for retry. The controlled native qualification
uses ExifTool 13.59 and platform-specific ffprobe distribution binaries; installed
product bundles and their notices remain part of release assembly.

## Low-interruption processing

The CLI is the primary import contract; the GUI wraps the same flags, presets, validation and result states. Import screens show the timezone/default-date policy once for a batch, allow per-item overrides and offer a details view rather than a repeated modal sequence. Configured copies/uploads, extraction, processing, supported speaker attribution and elected training proceed through durable jobs. New external routing or a meaningful destructive choice requires an explicit configuration/action; previously selected routing does not need repeated confirmation.

Capture and dates belong to admission, timed audio belongs to processing, and speaker associations expose the library-wide corpus. [Speaker models](voice-models.md) reuse this evidence when the user elects training. The [catalog schema](schema.md) preserves observations and model lineage independently of the selected storage/database adapters. [JSON contracts](contracts.md) define the import manifest and metadata payloads.

## Transcript and embedded admission

```sh
insonic media import recording.mp4 --transcript transcript.cueson.json --attribution auto
insonic transcript import transcript.vtt --record RECORDING_UUID --attribution native
insonic transcript import replacement.ass --record RECORDING_UUID --replace-transcript --attribution off
insonic transcript import --record RECORDING_UUID --legacy-sidecar
insonic transcript import --manifest transcripts.json
insonic media import recording.mkv --subtitle-language eng --subtitle-stream-index 2
```

Standalone transcript items use `record` plus `transcript`, creating no second media entry. An exact title must be unambiguous. A current transcript causes a successful skip by default; `--replace-transcript` elects replacement. Invalid explicit replacement leaves current state unchanged and never falls back to embedded text or recognition. Converting a managed legacy sidecar retires that redundant managed publication after current acceptance and lease reconciliation.

Exact packaged Cueson 1.0.0 and 1.1.0 documents validate under their original schemas, then undergo a lossless identity upgrade to 1.2.0. Current 1.2.0 documents validate directly. Unknown versions report the supported versions; input schema URLs are never fetched. Admission preserves native/source payloads, exact integers, cues and supplied assignments; JSON whitespace may be compacted by catalog serialization.

`--attribution auto` preserves any complete or partial assignments. Only a document with zero assignments derives recording-scoped untimed participation from structured native/heuristic speaker observations. `native` restricts derivation to native observations, and `off` preserves without deriving assignments. `diarize` requires `--diarization-model-id`, explicitly runs configured diarization and bypasses recognition for supplied text. Labels alone never create timed acoustic evidence or a known-person identity.

Media and transcript HTTP acquisition have separate credential IDs and budgets. Set `--transcript-credential-id`, `--transcript-max-bytes` (default and maximum 16 MiB), and `--transcript-timeout-ms` (default 120000, maximum 600000). Transcript bytes determine format independently of MIME and filename; HTML error bodies, incomplete downloads and invalid documents fail. `--transcript-format cueson|srt|vtt|ass|ssa` is an explicit hint validated by the selected parser. JSON/CSV item fields override batch defaults using the same shared contract. Advanced acquisition adapters are available through manifests/runtime input.

All embedded subtitle candidates remain inspectable in captured source-stream facts. An explicit transcript wins and reports unused embedded alternatives. Otherwise explicit index/language filters apply, then default disposition, non-forced disposition and stream order choose one supported text track. Alternatives are never concatenated. Image-only subtitles report an OCR requirement; burned-in captions require a separately elected OCR capability. Neither route silently runs an engine.

Standalone target selectors resolve to a recording ID and observed source/document revisions when queued. A delayed stale replacement fails instead of overwriting a later edit. Advanced manifest items can supply `expected_revision` and `expected_recording_revision` explicitly. Resubmit a new request to elect a changed target; accepted work retries reconcile their original receipts.
