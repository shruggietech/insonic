# S008 shared interface contracts

Version 0.0.0; original clocks are integer micro/milliseconds serialized as decimal strings when browser precision could be lost.

- graph.capabilities/status/publish/rebuild: selected backend identity, declared dialects/operations, current ordered checkpoints and pending work. Publish/rebuild is explicit; authority mutations enqueue invalidations automatically.
- evidence.extract MEDIA_ID: elected adapter/configuration, expected current recording/document, bounded window/overlap settings. Durable work returns reference-only acceptance.
- query.run/explain: normalized QuerySpec or declared native text/typed params; saved query ID/version optional. Common filters/ordering/pagination match graph-query schema.
- query.list/show/save: immutable query definitions, latest revision CAS, retained compatibility validation, no pinned_result.
- timeline.calendar: viewport/filters/page cursor across all library entries, selected date uncertainty and omitted/total counts.
- timeline.recording MEDIA_ID: current cues/voices/mapped people/segments/assertions, source-bound playback identities.
- views.show/save: separate query/view/layout metadata and CAS.
- search: CLI shorthand for normalized text/speaker/time search; query native requires explicit dialect/text input.

Every operation uses existing workspace/request envelope and public schema validation. Native unsupported syntax and mutation attempts fail before effects. Cross-workspace/oversized inputs fail. Existing playback tickets retain current media/document binding and expiration/revalidation.