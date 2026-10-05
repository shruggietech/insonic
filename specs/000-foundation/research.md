# Foundation research

## Sources refreshed on 2026-10-04

Current API/raw-source reads take precedence over stale indexed pages. No native product behavior is implied by this research.

| Topic | Primary source | Result |
| --- | --- | --- |
| Brand | [BrandBuilder v3.0.1](https://github.com/shruggietech/shruggie-brand/releases/tag/v3.0.1) | insonic 1.0.0 kit, checksums, manifest and enforcement bundle |
| Import patterns | [Glitchpad](https://github.com/shruggietech/glitchpad/blob/main/scripts/sync-brand-kit.mjs), [go-schedule](https://github.com/shruggietech/go-schedule/blob/main/scripts/brand-import/main.go) | Formal archive/receipt patterns with added latest-kit discovery |
| Docs | [Fragcap](https://github.com/h8rt3rmin8r/fragcap/blob/main/site/next.config.mjs), [Next static exports](https://nextjs.org/docs/app/guides/static-exports) | Markdown authority and static local assets |
| Subtitles | [Cueson architecture](https://github.com/shruggietech/cueson/blob/v1.1.0/docs/architecture.md), [schema](https://github.com/shruggietech/cueson/blob/v1.1.0/schema/releases/v1.1.0/cueson.schema.json) | CLI/schema boundary, real source envelope, millisecond timing, nonempty cue constraint |
| Graph | [Go binding](https://github.com/LadybugDB/go-ladybug), [concurrency](https://docs.ladybugdb.com/concurrency/) | Native/Cgo pair requires proof; one shared writer |
| Desktop | [Wails](https://wails.io/docs/introduction/), [Tauri](https://v2.tauri.app/concept/architecture/) | Go/Wails baseline with viable alternative |
| Local AI | [faster-whisper](https://github.com/SYSTRAN/faster-whisper), [whisper.cpp](https://github.com/ggml-org/whisper.cpp), [Community-1](https://huggingface.co/pyannote/speaker-diarization-community-1) | Replaceable workers and hardware-aware defaults |
| Workflow | [Spec Kit](https://github.com/github/spec-kit), [GitHub branch deletion](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-the-automatic-deletion-of-branches) | Actual templates, linked slices, human squash and automatic deletion |

Verified sibling activity: go-schedule and Glitchpad on 2026-09-29, the brand project and Fragcap on 2026-10-04. Their product requirements and heavy CI are not copied.

## Review resolutions

Publish immutable artifacts before committing catalog references. Compile terms before recognition. Keep repeated uploads and overlapping chunks as dependent evidence. Preserve uncertain identity/time. Generate real subtitle bytes before Cueson encoding and do not fabricate a zero-speech cue. Pin Ladybug native/core binding together in S001.

## Owner amendment research on 2026-10-04

[ExifTool format support](https://github.com/exiftool/exiftool/blob/master/README) covers audio/video metadata families beyond EXIF; [JSON extraction options](https://github.com/exiftool/exiftool/blob/master/html/exiftool_pod.html) require duplicate-instance handling. [QuickTime tag documentation](https://github.com/exiftool/exiftool/blob/master/html/TagNames/QuickTime.html) warns that cameras can record local values in conventionally UTC fields. Preserve raw observations and use a recorded interpretation policy. The [IANA database](https://www.iana.org/time-zones) supplies historical zone rules. These sources support the new import contract; no media extractor was executed.

The [storage and catalog schema](../../docs/v0.0.0/schema.md) records the researched S3/PostgreSQL/ArcadeDB boundaries and links their primary sources. Owner-approved alternate backends replace the earlier recommendation against a network catalog. Optional speaker training replaces the earlier later-direction wording; no particular synthesis engine or mandatory enrollment review is implied.
