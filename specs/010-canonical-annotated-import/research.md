# S010 research and decisions

## 2026-10-09: storage policy

Decision: one grouped .mka with FLAC tracks, stereo-inclusive FLAC override, lossy mono MP3 passthrough/V0 encoding, format-only normalization without automatic DSP. All owner answers are recorded in spec.md.

Rationale: preserve independent tracks and the literal multichannel requirement. Alternatives were native-FLAC-only groups, a greater-than-two-channel override and FLAC substitution for other lossy codecs; none was adopted without owner agreement.

## 2026-10-09: admission authority

Decision: refactor current recording publication into a reusable transaction operation and add atomic admission with source/document revision fences.

Rationale: library admission currently commits before recording processing. Two commits could leave accepted audio without the explicitly required transcript after failure. Reuse current mapping/evidence invalidation instead of inventing another authority.

## 2026-10-09: document identity and compatibility

Decision: direct import bypasses destructive assembly. Package exact historical schemas and translate known identity fields only after exact schema/semantic validation where field compatibility is demonstrated.

Rationale: AssembleDocument deletes incoming speaker_attributions and current validators narrow upstream IDs to UUIDs. Global catalog IDs stay UUIDs; bounded local attribution IDs follow Cueson and remain recording-scoped.

Primary sources: [Cueson 1.2.0](https://github.com/shruggietech/cueson/tree/v1.2.0), [consumer contract](https://github.com/shruggietech/cueson/blob/v1.2.0/docs/consumer-speakers.md), [immutable schemas](https://github.com/shruggietech/cueson/tree/v1.2.0/schema/releases).

## 2026-10-09: canonical facts and playback

Decision: canonical digest/size/publication occupy the current library identity fields; additive facts preserve captured source identity and ordered source-to-canonical stream maps. Probe canonical output instead of applying source stream indices to remuxed bytes.

Rationale: processing/playback current digest checks then remain meaningful. Audio preparation stays temporary and canonical sample/detail policy is independent of a selected model.

Primary sources: [FFmpeg stream selection](https://ffmpeg.org/ffmpeg.html#Stream-selection), [Matroska FLAC mapping](https://www.matroska.org/technical/codec_specs.html#a_flac), [FLAC constraints](https://www.xiph.org/flac/faq.html).

## 2026-10-09: native packaging

Decision: verify MP3 encoder support in every pinned package and source builder. Current macOS source build enables AAC/FLAC/PCM only, so MP3 V0 requires a pinned actual encoder rather than pretending the binary supports it.

Rationale: passing fixtures on a different host does not prove native package capability. Keep model inference outside CI and measure affected job duration.

## Remaining tracking

Full explicit audio replacement and bulk legacy-audio migration remain #30/#33. Model aliases, rosters, training and release promotion remain #34/#35/#15/#14. S010 may close #31/#32 only after acceptance is demonstrated.