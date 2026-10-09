# Media and provenance

## Originals and acquisitions

Audio and video share one `MediaEntry` catalog model. A media entry represents a library item; a `MediaAsset` represents a particular byte sequence and selected streams. Content hashes identify bytes, while stable entry IDs retain user metadata and processing history. The same bytes can be associated with several entries without becoming independent evidence automatically.

New imports capture original metadata and subtitles before storing canonical audio in the configured filesystem/S3 artifact store. Mono lossy codecs use high-quality MP3 V0, or packet-copy suitable existing MP3. Stereo and larger channel layouts use FLAC. Multiple independent audio tracks remain one recording in an audio-only Matroska `.mka` asset with FLAC tracks, preserving ordering, language, disposition, channels and source offsets. Existing FLAC tracks are copied when suitable. Rates unsupported by MP3 use FLAC rather than implicit resampling. Floating-point lossless PCM that cannot be represented exactly by FLAC fails explicitly; integer PCM retains its sample precision.

Canonicalization means format standardization. It does not elect loudness adjustment, denoising, enhancement, upsampling, channel mixing or silence removal. Model-specific mono PCM preparation and browser previews are disposable derivatives. Original digests and rational track-clock maps remain provenance alongside the canonical byte identity.

Owner-owned input files remain untouched. New managed storage retains canonical audio, captured source facts and one current embedded Cue JSON document when a transcript is selected. It does not retain the original video, a redundant native subtitle file or the original local input locator. Accepted work replaces its item input with a receipt-bound marker, so retry can reconcile after the inputs are removed. Stable nonsecret remote acquisition locators remain provenance; credentials remain external references. Legacy copied/reference entries remain readable and relocatable. New reference-mode admission is unsupported. Audio replacement is explicitly elected on a stable recording; format changes alone do not require bulk conversion of pre-release media.

Never edit an accepted asset in place. Content hashes identify bytes; stable entry IDs retain metadata and processing history. Hash equality supports deduplication, while titles and durations are comparison leads rather than identity proof. [Import, metadata and dates](ingestion.md) defines admission and date policy.

## Audio derivatives

Video audio extraction, resampling, channel selection, normalization and inference chunks create derived assets. Each records its parent hash, source stream, channel layout, transform specification, tool and version, sample format, duration and mapping to original media time. Default transcription input can be optimized mono PCM at the chosen model's supported sample rate. Do not mix channels when that would destroy useful separation; let the user select or preserve channels.

```mermaid
flowchart TB
  Original[Immutable canonical audio or readable legacy source] --> Probe[Streams, duration and byte identity]
  Probe --> Audio[Selected full-duration audio derivative]
  Audio --> ASR[Transcription input or chunks]
  Audio --> Diarization[Acoustic speaker segmentation]
  ASR --> Mapping[Map intervals to original media clock]
  Diarization --> Mapping
  Mapping --> Transcript[Correlated transcript and voice intervals]
  Mapping --> Segments[Speaker-tagged audio segment catalog]
  Segments --> Corpus[Library-wide corpus linked to speaker identities]
```

Full-duration extraction retains silence and sequence by default. A chunk has an offset and bounded source interval. Silence removal or concatenation requires a piecewise time map; a single offset is insufficient. Unsupported time-remapping transforms fail explicitly rather than emitting misleading timestamps. Variable-frame-rate video uses presentation timestamps, not frame-number division by an assumed rate.

Speaker-tagged audio consumers reference the current recording/document/cue/local speaker token. Resolve timed intervals and channel/source mapping from that authority instead of persisting another assignment list. Untimed participation has no exact clip interval. Managed clips are derived current data; replacing a document or correcting its known-speaker mapping invalidates affected corpus preparation and physically retires removed outputs through recoverable cleanup. The corpus spans the library, and optional training follows [speaker audio and voice models](voice-models.md).

## Time and correlation

Store media-relative integer microseconds internally; convert to Cueson's integer milliseconds using a documented rounding rule at the subtitle boundary. Preserve higher precision in the insonic correlation sidecar. Source interval boundaries must remain ordered and within the probed duration, with an explicit codec-tolerance policy. An export must not claim precision greater than its format provides.

Keep recording time, publication time, retrieval time, import time and processing time separate. Recording dates support earliest/latest bounds, precision, time zone, basis and supporting source. File modification time does not become recording time. Unknown dates remain unknown and appear in the timeline's undated group. A derived file inherits its relationship to the original clock, not a newly invented recording date.

An import-supplied timestamp without a zone uses the captured local timezone by default, with UTC/IANA/offset overrides. Current embedded observations and owner date selections remain distinct records. Refresh replaces computed metadata and retires the superseded reports; date correction preserves owner observations and the current source report. Date-only input retains its precision; corrections update calendar placement without altering media-relative timing or the original metadata. See the complete [date policy](ingestion.md#origination-dates-and-timezones).

Transcript admission names an existing recording by UUID or unambiguous exact title, or accompanies a new media item. Explicit input wins over embedded alternatives. Supplied document timing remains authoritative; admission does not silently offset, scale or replace it. Native bytes remain recoverable from the embedded Cueson source envelope.

## Retention and export

Canonical audio, captured source facts and accepted current evidence are durable. Legacy originals remain available under their existing references. Disposable decode caches are regenerable. Downloaded base models use digest-verified workspace artifact storage and exact manifests, independently of trained speaker model lineage. Trained speaker model versions and checkpoints use durable workspace artifact storage. A deletion presents affected entries/files and distinguishes removing an index entry from deleting managed bytes; an explicit deletion action can proceed without a separate repeated approval dialog. Referenced external files are never deleted by removing a library entry.

A portable workspace export includes a manifest, content-addressed managed assets or explicit missing external references, current raw metadata and selected date observations, one current embedded document per recording, external speaker/term catalogs, reference-only segment/dataset/model lineage, pipeline definitions and evidence metadata. Model binaries and corpus clips can be included or left as explicit remote/missing artifact references according to export options. It excludes credentials. Import validates portable paths and identities; it cannot rely on the exporting machine's absolute paths or provider credentials.

## Explicit audio replacement

Target an existing recording by UUID or unique exact title. Without `--replace-audio`, admission returns a successful skip before opening or downloading the candidate. Replacement retains the recording ID and canonical codec/container, track separation and exact source-clock policy. Source facts are captured before transformation. The new source and elected current transcript/roster effects commit together; a failed, cancelled or stale attempt preserves the accepted recording.

When a current transcript exists, choose `--existing-transcript keep|clear|replace`. Keep requires `--transcript-applies`, preserves exact document bytes and validates known cue/consumer intervals in the new source clock. Unknown audio bounds remain diagnosed uncertainty. Clear publishes `untranscribed` with a null document, without asserting silence. Replace requires a validated explicit or selected embedded transcript. Without an existing document, selected transcript input implies replace; otherwise replacement clears. Processing follows only a separately elected stage.

Changed audio invalidates prior acoustic mappings, current clips and dependent corpus preparation even when the kept document digest is identical. Current playback, queued processing and extraction are fenced against superseded authority. Old managed publications retire through recoverable reference/lease checks after acceptance; owner inputs and shared current bytes remain intact. Date selections survive unless the import elects a new date.

```sh
insonic media import replacement.wav --record RECORDING_ID --replace-audio --existing-transcript keep --transcript-applies --existing-roster retain --json
insonic media import replacement.mkv --record RECORDING_ID --replace-audio --existing-transcript replace --transcript captions.cueson.json --existing-roster clear --json
```

If a roster header exists (including an explicitly empty roster), audio replacement requires `--existing-roster retain|clear`. Transcript-only changes preserve the roster. A new membership list is initial-admission input; edit an existing roster through the shared [roster commands](speakers.md#declared-recording-rosters).
