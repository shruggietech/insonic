# CLI and desktop experience

## Shared operations

The application is CLI-first. Every core capability has an automation-ready CLI contract, and the GUI installs the CLI and wraps equivalent shared-runtime operations. Both use the same workspace IDs, job states, pipeline definitions, query parameters and errors. No core behavior lives only in GUI code. Shared capability and release matrices record which client exposes each operation.

Advanced or experimental CLI capabilities may ship before their GUI controls. Each release documents that lag in its capability matrix, including the CLI command, current GUI availability and its follow-up outcome. This exception does not turn mature core operations into permanent CLI-only features. Speaker-model discovery and retrieval begin with CLI contracts; desktop controls follow the shared operation model.

CLI groups are `workspace`, `media`, `pipelines`, `jobs`, `speakers`, `terms`, `search`, `query`, `export`, `models`, `settings` and `doctor`. Human output is concise; `--json` provides versioned machine-readable records, diagnostics go to stderr, and exit statuses distinguish failure and partial completion. Noninteractive invocations never prompt. Secret entry uses a protected input channel, not command-line values.

The source-buildable desktop exposes Library, Jobs, Pipelines, Speakers, Terms, Models and Settings screens. The native bridge uses the same workspace/runtime requests and protected credential channel as the CLI. Library details support captured metadata, date corrections, reference relocation, current recording assembly/processing, inspection, rerun, native export and known-speaker mappings. See [subtitles](subtitles.md), [pipelines](pipelines.md), [credential commands](secrets.md), [downloaded models](models.md) and [import](ingestion.md) for shared validation and executable operations.

| Capability | CLI | Desktop |
| --- | --- | --- |
| Create/open workspace, import/list/inspect media, metadata/date correction and relocation | Implemented | Dedicated Library controls and native dialogs |
| Current Cueson assembly, configured processing, rerun, export and speaker mappings | Implemented | Library detail controls and current cues |
| Saved pipelines and context, speaker identities/aliases/current evidence, terminology | Implemented | Pipelines, Speakers and Terms controls |
| Durable work inspection, cancellation, retry and recovery | Implemented | Jobs controls; window closure preserves work |
| Downloaded model register/acquire/list/show/verify/materialize and credentials | Implemented | Models and Settings controls |
| Media/processing tool configuration and appearance | Shared settings operations | Revision-checked Settings forms |
| Original/preview playback and current cue/span seeking | Shared playback contracts | Native audio/video controls |
| Low-level artifact retention/leases, catalog snapshots, profile migration and detailed raw bundles | Advanced commands | Readable profiles; advanced operations remain in the CLI |
| General text/graph exploration, query assistance and speaker-model training | Delivery contracts | Subsequent implementation |

The matrix describes implemented interfaces, not an official product release. Dedicated profile migration controls follow catalog migration/backup delivery; saving an appearance or tool setting does not replace storage/catalog configuration.

The following examples illustrate the product contracts. General exploration and training require later implementation; consult each capability page for its current executable command syntax:

```sh
insonic workspace create --path PATH
insonic media import FILE --copy
insonic media import FILE --copy --originated-at "2024-06-08T14:30:00" --timezone America/New_York
insonic media import FILE --copy --originated-on 2024-06-08
insonic media import --manifest imports.json
insonic media attach-subtitle MEDIA_ID SUBTITLE_FILE
insonic jobs list --json
insonic search "sample phrase" --json
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

Failed or interrupted supplied-turn assembly requires resubmitting its transient inputs through the Library assembly controls. Jobs explains this action; generic retry does not reconstruct unsaved turn inputs. Durable import, processing and model-acquisition work retains its shared retry behavior.

Playback uses the original or a documented preview proxy. Selecting a cue, assertion or graph evidence opens its correlated original-media position. If a referenced original moved, the application asks for relocation and verifies the asset identity before reconnecting it.

Implemented Library playback uses private native tickets scoped to the selected workspace, media revision and current recording/document. The desktop starts a playback-only HTTP transport on 127.0.0.1 with an ephemeral port and closes it on exit. This supports native media engines that cannot consume Wails custom-scheme media. Its handler supports ranged GET/HEAD requests, checks current authority and renews materialization leases during streaming. Host and browser Origin checks limit requests; ticket URLs expose no native paths and the endpoint provides no general-purpose CORS API. The mounted player also checks every five seconds so a replacement from another client stops already buffered playback and clears obsolete cue selections. Closing playback or selecting another workspace releases its handle; a changed original, replacement recording or expired handle requires a fresh request. Files are never served by accepting an arbitrary path from the webview.

Unsupported media and nonzero source clocks receive a bounded temporary WAV or MP4 preview made by the configured hash-verified FFmpeg tools. Original bytes remain immutable. Preview responses identify served and source digests separately and carry the measured difference between original and preview timelines used when seeking. Zero-clock compatible originals play directly. Current cue clocks reach JavaScript as exact decimal strings; raw capture pages retain byte-exact JSON rather than rounding nanosecond integers through JavaScript numbers. Cues and speaker intervals always resolve against current embedded Cueson authority.

## Native package layouts

The package builder creates Windows x86-64 ZIP, Linux x86-64 tar.gz and macOS ARM64 application ZIP layouts. Each includes sibling GUI/CLI executables, pinned Cueson and schema, metadata/media companions, native libraries, offline help, dependency notices and a SHA-256 inventory. Companions are discovered relative to the installed executable and verified before use. An explicit workspace tool configuration takes precedence; malformed configuration reports an error rather than silently falling back. Installation paths are not written into portable workspace settings.

Settings displays effective packaged tools read-only. Selecting “Edit explicit workspace media tools” or “Edit explicit workspace processing tools” opens separate override inputs without copying installation paths. Save remains disabled while editing is off; turning editing off leaves an existing override in place. The matching “Reset … to installed defaults” action removes the override with a revision check and resumes installation-relative discovery. The shared CLI supports the same reset through `settings set --input reset.json`, with a section, its current revision and `"value": null`.

Windows requires WebView2 and the Microsoft Visual C++ runtime. Linux requires GTK 3, WebKitGTK 4.1, Perl and GStreamer base/good/bad/libav decoder packages for webview audio/video playback (`gstreamer1.0-plugins-base`, `gstreamer1.0-plugins-good`, `gstreamer1.0-plugins-bad` and `gstreamer1.0-libav` on Ubuntu). Its installation helper installs the application launcher. The macOS application targets macOS 13.3 or newer. Other architecture combinations have no executed package claim. Local inference Python environments and model weights remain separately configured downloads.

Qualification extracts packages into relocated paths containing spaces and exercises CLI, media import, Cueson, offline help and the actual native webview. Package construction is separate from release publication: these archives are unsigned and not promoted as official downloads. Release delivery must complete signing/notarization and the corresponding-source distribution for GPL-enabled companion builds before publishing binaries.

## Accessibility and offline help

Support keyboard navigation, readable contrast, reduced motion, scalable text, screen-reader labels and tabular alternatives for timeline/graph content. Automatic light/dark styling follows the upstream brand tokens and user preference. Graph physics can be paused and a layout saved.

Help opens the documentation-only bundle from `site/offline/` for the installed application version. The public product landing page is part of the website export in `site/out/` and is excluded from installed help. It works without a network connection and clearly states the version. Online help offers the matching version first and latest docs as a deliberate navigation choice. The GUI cannot imply that a feature documented for a newer release exists in an older installation.

The documentation website and bundled offline help follow the operating-system theme by default. A sun/moon toggle switches between light and dark mode and remembers the choice when browser storage is available. This setting controls documentation appearance independently of the desktop application's preference.
