# Research: S063

## Exact source clock and bounded extraction

Decision: count actual native samples and reuse one continuous resample stream per track, with backpressure and bounded window buffers. Rationale: current Prepare copies/materializes and decodes the entire source; wrapping it per window repeats I/O and retains duration-sized PCM. Alternatives: metadata-derived duration and independent seek/resample were rejected due to compressed padding, offsets, and phase differences. Use artifact range reads and a bounded chunk-digest inventory to verify later decoding reads before they reach the decoder, avoiding a source-sized materialization. Retain lease/reference fences and hidden process supervision.

## Durable checkpoint authority

Decision: typed ProcessingPlan and ProcessingWindowCheckpoint domains, fenced commits and operation receipt proofs. Rationale: Work.Result is transient progress and RetryWork clears it. Alternatives: embedding output in work progress or unverified scratch manifests lose authority across retry/restore. Ready rows pin verified publications; final acceptance/replacement retires them through existing cleanup and backup selection.

## Word evidence and resource recovery

Decision: optional mapped-relative word observations from local/hosted recognition, same-track temporal plus lexical overlap matching, explicit ambiguity, final Cueson cue-ID provenance stored outside the document. Rationale: current SRT alone discards word evidence; upstream Cueson forbids arbitrary extensions. Distinct repeated text and independent track speech must survive.

Decision: preserve existing CUDA policy with at most one compatible pyannote batch reduction before CPU fallback. Whisper is unbatched; changing beam/window/model would change semantics and is rejected. Actual device and reductions belong per-attempt evidence rather than plan identity.

## Research sources

- Current processing/audio.go, processing results/device policy, app processing/election, catalog work/speaker checkpoints/backup, artifact materialization and process supervision.
- [faster-whisper word timestamp implementation](https://raw.githubusercontent.com/SYSTRAN/faster-whisper/v1.2.1/faster_whisper/transcribe.py).
- [pyannote batch controls](https://raw.githubusercontent.com/pyannote/pyannote-audio/4.0.7/src/pyannote/audio/pipelines/speaker_diarization.py).

No unresolved architecture questions remain. Implementation details are constrained by the contracts and acceptance tests.

## Verified MP3 gapless input behavior

Pinned FFmpeg 7.0.2 clears Xing frame count when unknown pipe byte length fails its plausibility check, losing final padding trim. The bounded source reader retains only the first MPEG frame after skipping ID3 tags, verifies Xing/LAME endpoints against observed native decoded samples, and trims before continuous resampling. Native byte comparisons cover variable/constant bitrate and multiple sample rates. Primary implementation reference: [FFmpeg MP3 demuxer](https://github.com/FFmpeg/FFmpeg/blob/n7.0.2/libavformat/mp3dec.c#L164-L191).
