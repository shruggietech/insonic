# CLI and desktop experience

## Shared operations

The application is CLI-first. Every core capability has an automation-ready CLI contract, and the GUI installs the CLI and wraps equivalent shared-runtime operations. Both use the same workspace IDs, job states, pipeline definitions, query parameters and errors. No core behavior lives only in GUI code. Shared capability and release matrices record which client exposes each operation.

Advanced or experimental CLI capabilities may ship before their GUI controls. Each release documents that lag in its capability matrix, including the CLI command, current GUI availability and its follow-up outcome. This exception does not turn mature core operations into permanent CLI-only features. Speaker-model discovery and retrieval begin with CLI contracts; desktop controls follow the shared operation model.

CLI groups are `workspace`, `media`, `pipeline`, `jobs`, `speakers`, `terms`, `search`, `query`, `export`, `models`, `settings` and `doctor`. Human output is concise; `--json` provides versioned machine-readable records, diagnostics go to stderr, and exit statuses distinguish failure and partial completion. Noninteractive invocations never prompt. Secret entry uses a protected input channel, not command-line values.

The implemented source-buildable clients expose workspace/runtime, catalog,
artifact, credential, downloaded-model, media-admission and real-work operations.
The native bridge exposes `Operate` and `Credential` using those same contracts.
Full media/settings GUI controls follow in the desktop delivery outcome.
See [credential commands](secrets.md), [downloaded models](models.md) and
[import](ingestion.md) for the executable operations.

The following broader product commands are target contracts. Pipeline, speaker,
search, training and installable GUI workflows require later implementation:

```sh
insonic workspace create --path PATH
insonic media import FILE --copy
insonic media import FILE --copy --originated-at "2024-06-08T14:30:00" --timezone America/New_York
insonic media import FILE --copy --originated-on 2024-06-08
insonic media import --manifest imports.json
insonic media attach-subtitle MEDIA_ID SUBTITLE_FILE
insonic pipeline run MEDIA_ID --preset local
insonic jobs list --json
insonic search "sample phrase" --json
insonic speakers segments list --speaker SPEAKER_ID
insonic models dataset create --speaker SPEAKER_ID --preset speech-clean
insonic models train --dataset DATASET_ID --pipeline PIPELINE_ID
insonic models list --speaker SPEAKER_ID
insonic models show MODEL_VERSION_ID
insonic models fetch MODEL_VERSION_ID --output MODEL_DIRECTORY
```

`--originated-at` accepts a date and time; `--timezone` accepts an IANA zone, UTC or a numeric UTC offset. An unzoned supplied time uses the captured local IANA zone for that import. `--originated-on` records date-only precision and does not invent midnight or an exact instant. CSV and JSON batch manifests carry per-item source paths, date precision, zone and metadata overrides; command-level values are defaults and item values take precedence. The default date precedence is owner-supplied value, valid zoned embedded value, explicitly timezone-interpreted embedded value with its assumption recorded, then unknown. File modification time is never silently treated as origination time. `--date-precedence` selects a saved rule when another ordering is useful.

Repeated daylight-saving wall times accept an explicit offset or `--dst-fold earlier|later`. Nonexistent wall times require a corrected date or explicit `--dst-gap shift-forward` policy. The media is still admitted with that date observation marked unresolved until disambiguated. Batch jobs continue with unaffected items; noninteractive commands return a partial-completion status and actionable JSON without prompting. Missing or unusable embedded dates never block admission. Media inspection exposes the chosen recording time, precision, zone, source and competing metadata observations so corrections are understandable and reversible. See [import, metadata and dates](ingestion.md).

## First-run experience

1. Choose the workspace location.
2. Add audio or video.
3. Capture metadata and any supplied origination dates.
4. Attach subtitles if available.
5. Choose local or connected processing.
6. Explain the required model downloads and storage.
7. Process with progress, pause and cancellation controls.
8. Explore transcripts, speakers, the timeline and queries.

Default screens use task names such as Transcription, Speakers and Query assistance. Explain models and servers only when the user needs to choose one. Recommend a hardware-appropriate local preset, show download size, and allow an override. Provider setup tests configuration with a small appropriate request and reports whether it succeeded; it never uploads library contents just to test a key.

Selecting a preset and its routing preferences authorizes its routine operations. Imports, derivative generation, configured speaker attribution and jobs proceed without per-item approval dialogs. Electing a speaker-training workflow authorizes its saved selection and destination rules; corpus qualification and model validation are automated checks. Users can inspect or override results, and an error pauses only the affected work. New external destinations, unresolved dates and previously unauthorized destructive effects explain the concrete decision that needs input.

## Desktop surfaces

The Library supports audio and video in the same list with import/indexing state. Media details contain playback, captured metadata, date provenance, transcript tracks, speaker intervals, related derivatives and processing history. Raw metadata remains inspectable beside normalized facts. Jobs show current stage, elapsed time and the next useful action on failure. Speakers and Terms expose the catalogs without requiring schema knowledge; speaker details include source-linked audio corpora and revisions. Explore contains text search, the calendar/media timeline and saved graph queries. Settings contains workspaces, storage/catalog/graph adapters, model storage, providers and pipeline presets.

AI query assistance shows proposed read-only queries by default. An explicit `--run-if-valid` option or saved auto-run preference can execute suggestions after parsing, read-only validation and configured limits. This preference avoids repeated confirmation while retaining direct queries when assistance is disabled. Invalid or unsupported suggestions report their error without execution.

Progress distinguishes decoding, model download, inference, subtitle publication and graph indexing. Closing the window does not silently cancel durable jobs. Users can keep work running, pause future work or cancel selected jobs. Runtime restarts show interrupted attempts explicitly and offer resumable work.

Playback uses the original or a documented preview proxy. Selecting a cue, assertion or graph evidence opens its correlated original-media position. If a referenced original moved, the application asks for relocation and verifies the asset identity before reconnecting it.

## Accessibility and offline help

Support keyboard navigation, readable contrast, reduced motion, scalable text, screen-reader labels and tabular alternatives for timeline/graph content. Automatic light/dark styling follows the upstream brand tokens and user preference. Graph physics can be paused and a layout saved.

Help opens the documentation-only bundle from `site/offline/` for the installed application version. The public product landing page is part of the website export in `site/out/` and is excluded from installed help. It works without a network connection and clearly states the version. Online help offers the matching version first and latest docs as a deliberate navigation choice. The GUI cannot imply that a feature documented for a newer release exists in an older installation.

The documentation website and bundled offline help follow the operating-system theme by default. A sun/moon toggle switches between light and dark mode and remembers the choice when browser storage is available. This setting controls documentation appearance independently of the desktop application's preference.
