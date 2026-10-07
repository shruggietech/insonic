# Import, metadata and dates

## A definite import workflow

Import accepts local audio/video, a batch manifest or an acquisition adapter's output. The default is a managed copy into the configured artifact store. Keep in current location is available for local references. A saved import preset supplies the destination, timezone policy, date precedence and optional processing pipeline. A configured batch runs without approving every file or stage.

Capture metadata from the original bytes as soon as a stable readable source is available, before transcoding, resampling, remuxing, stripping tags or producing a proxy. For remote sources, retain available acquisition metadata before download and extract embedded metadata from the downloaded original before transformation. A source that has already lost metadata cannot have those missing fields reconstructed as facts.

1. Accept local files, a batch manifest or an acquired original.
2. Read stable original bytes and compute their identity.
3. Capture embedded metadata and stream facts.
4. Publish the original and the current metadata reports.
5. Commit media records, observations and the selected-date state.
6. Make the library entry available immediately.
7. Return durable work and per-item admission results. Processing presets remain recorded inputs until downstream processing is implemented; no unexecuted processing job is reported as complete.

The admission receipt binds the original digest, byte size, source locator, acquisition/import times, metadata report digests, extractor versions/options, probe results, effective date policy and selected artifact locations. Detect a source changing during copy or extraction and retry from stable bytes instead of joining a report to the wrong asset. Derived processing starts only after original admission and the capture attempt are recorded.

## Metadata preservation

EXIF is one supported metadata family; audio/video can also carry XMP, ID3, RIFF/BWF, QuickTime and other container tags. Capture all available families, container-level tags and per-stream facts, rather than requiring every file to contain EXIF. [ExifTool](https://github.com/exiftool/exiftool/blob/master/README) is the default embedded-metadata extractor; [ffprobe](https://ffmpeg.org/ffprobe.html) supplies complementary container, stream and codec facts. Release packages pin these tools and include their licensing notices.

Persist the current raw extractor report and separate typed normalized observations. A refresh replaces the current computed metadata and physically retires superseded reports after active materialization leases are released. Keep the source media and supplied subtitle bytes. Preserve tag group/path, duplicate instance, original value, units, precision and warnings. ExifTool JSON can suppress identical tag names unless instance-qualified groups are requested; the extraction contract must retain duplicates and structured values. [Official extraction documentation](https://github.com/exiftool/exiftool/blob/master/html/exiftool_pod.html) describes these options. Unknown or binary fields must remain recoverable from the preserved original; a bounded report can link retained payload artifacts rather than silently truncate them. Report truncation and coverage explicitly.

For a local reference import, retain the extracted report and available metadata payload artifacts immediately even though media stays outside managed storage. Later source loss leaves those captured observations intact; it cannot guarantee recovery of an unparsed field that existed only in the missing original. State that availability limit, and preserve the managed-copy default for users who want durable original bytes.

Capture states are `captured`, `no-embedded-metadata`, `partial`, `unsupported` and `failed`. Only a successful extraction with no embedded fields establishes `no-embedded-metadata`. Unsupported extraction and errors retain the original plus the capture attempt and allow retry. They do not reject otherwise usable media or claim all metadata was captured. If a limit prevents complete extraction, keep the input item available with a visible partial state.

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

A JSON manifest carries a version, batch defaults and ordered items. CSV supports the same flat per-item fields. Each item has a source, optional subtitle attachment, origination timestamp/date, timezone and preset override. Sources and subtitle paths are resolved relative to the manifest location unless explicitly absolute. Do not put credentials in manifests; use configured credential IDs. Example:

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

Import and refresh return a durable `work_id`. Inspect completion through
`work show`; large batches expose references through
`work results WORK_ID --after-ordinal 0 --limit 100`. Continue from the returned
`next_ordinal`. `media list`, `models list` and `work list` return summary pages
with `next_id`; pass that UUID with `--after` for the next page. Default and
maximum page size are 100. `media raw` returns current capture-bundle paths with
artifact leases rather than placing large reports in IPC frames. Oversized
normalized metadata also returns these reports with `metadata_inline: false`.
Release each lease after reading using `artifacts lease-release`.

JSON date input can preserve approximate `originated_earliest` and
`originated_latest` bounds as exact `{iso, unix_ns}` UTC instants. Either bound
may be absent; two bounds must be ordered and their text and integer values must
agree. A bounded observation has range precision and no invented exact recording
instant. Per-item bounds replace batch bounds and cannot be combined with an
exact `originated_at` or `originated_on` value.

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
